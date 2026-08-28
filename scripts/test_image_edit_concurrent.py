#!/usr/bin/env python3
"""
并发测试 /v1/images/tasks/edits 接口
发送10个并发任务，持续轮询每个任务的状态直到全部完成。

用法:
    python3 scripts/test_image_edit_concurrent.py \
        --url http://localhost:9090 \
        --key YOUR_API_KEY \
        --model gpt-image-2 \
        --image path/to/image.jpg \
        [--concurrency 10] [--n 1] [--size 1024x1024] [--poll-interval 3]
"""

import argparse
import base64
import json
import sys
import time
import threading
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime
from pathlib import Path

try:
    import urllib.request
    import urllib.error
except ImportError:
    pass

PROMPT = (
    "具体生成任务：根据提供的产品图和模特图，让模特图中的人物真实、自然地佩戴产品图中的微信图片_2026-07-05_212433_351_cropped（戒指）。"
    "产品图决定产品外观，模特图决定人物外观。最终只输出一张完整的真人佩戴照片，不得输出拼图、参考图边框、空白区域或说明文字。\n\n"
    "根据提供的“产品图”和“模特图”，生成一张真实自然的真人佩戴产品照片。产品尺寸偏小且符合人体佩戴逻辑：产品尺寸必须偏真实、"
    "戒指戴在手指上必须清晰可见但视觉尺寸偏小，精致，符合真人佩戴比例。戒指保持真实戒宽。禁止为了展示产品而过度放大。"
    "产品图是唯一标准：产品图片为最高优先级参考，不是“类似款”，不允许参考数据库相似款，不允许自由设计、优化款式或替换产品元素。"
    "材质与工艺质感真实：保持真实金属反射、真实宝石折射、真实贝母纹理、真实电镀层和真实镶嵌工艺。"
    "严禁塑料感、玩具感、橡胶感、3D建模感、CG渲染感，钻石和金属不能过度发光。"
    "iPhone17拍摄真实感：整体画面必须像真实博主用 iPhone17 随手拍摄，不像商业棚拍，不像珠宝广告图。"
    "允许轻微自动曝光、白平衡偏差、HDR痕迹、手机锐化、轻微噪点和JPEG压缩感。"
    "人体结构与画面错误限制：严禁多手、多手指、多耳朵、多锁骨、错误关节、错误镜像、穿模、产品悬浮、随机文字、水印、乱码、重复纹理和不合理肢体结构。"
    "产品不可改款：产品必须严格保持原有结构、外轮廓、比例、层级、连接方式、开口位置、链条结构、镶嵌方式和雕刻纹理、戒指变宽戒，"
    "严禁新增、删除或替换任何设计元素。真实佩戴逻辑：产品必须真实佩戴在人体对应位置。"
    "戒指自然贴合手指，不允许悬浮、嵌入皮肤、脱离人体或佩戴位置错误。\n\n"
    "拍摄风格要求：清冷韩系通勤风。整体色调以白色、奶油色、浅灰、炭灰、柔和米色为主；"
    "模特妆感轻、发型整洁、神情自然克制。场景为现代咖啡店、办公楼大厅、商场走廊、书店或电梯。"
    "自然日光与干净室内光混合，画面安静高级，像真实 iPhone 日常随拍，不要商业硬广感。"
)

# 状态显示颜色（ANSI）
COLORS = {
    "pending":    "\033[33m",   # 黄
    "processing": "\033[34m",   # 蓝
    "completed":  "\033[32m",   # 绿
    "failed":     "\033[31m",   # 红
    "reset":      "\033[0m",
    "bold":       "\033[1m",
    "dim":        "\033[2m",
}

def colorize(text: str, color: str) -> str:
    return f"{COLORS.get(color, '')}{text}{COLORS['reset']}"

def http_post_json(url: str, headers: dict, body: dict) -> tuple[int, dict]:
    data = json.dumps(body).encode("utf-8")
    req = urllib.request.Request(url, data=data, headers=headers, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return resp.status, json.loads(resp.read())
    except urllib.error.HTTPError as e:
        body_bytes = e.read()
        try:
            return e.code, json.loads(body_bytes)
        except Exception:
            return e.code, {"error": body_bytes.decode("utf-8", errors="replace")}

def http_get_json(url: str, headers: dict) -> tuple[int, dict]:
    req = urllib.request.Request(url, headers=headers, method="GET")
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return resp.status, json.loads(resp.read())
    except urllib.error.HTTPError as e:
        body_bytes = e.read()
        try:
            return e.code, json.loads(body_bytes)
        except Exception:
            return e.code, {"error": body_bytes.decode("utf-8", errors="replace")}
    except (urllib.error.URLError, OSError) as e:
        # 网络抖动（DNS 失败、连接超时等）——返回哨兵值，保持原状态继续轮询
        return 0, {"_network_error": str(e)}

def submit_task(idx: int, base_url: str, headers: dict, image_b64: str, model: str, size: str, n: int) -> dict:
    """提交一个编辑任务，返回任务信息。"""
    url = f"{base_url.rstrip('/')}/v1/images/tasks/edits"
    body = {
        "model": model,
        "prompt": PROMPT,
        "size": size,
        "n": n,
        "image_base64s": [image_b64],
    }
    status, resp = http_post_json(url, headers, body)
    if status in (200, 202):
        return {"idx": idx, "id": resp.get("id"), "status": resp.get("status", "pending"), "error": None, "submitted_at": datetime.now().isoformat()}
    else:
        err_msg = resp.get("error", {})
        if isinstance(err_msg, dict):
            err_msg = err_msg.get("message", str(resp))
        return {"idx": idx, "id": None, "status": "submit_failed", "error": f"HTTP {status}: {err_msg}", "submitted_at": datetime.now().isoformat()}

def poll_task(task_id: int, base_url: str, headers: dict) -> dict:
    """轮询单个任务状态，返回最新任务数据。"""
    url = f"{base_url.rstrip('/')}/v1/images/tasks/{task_id}"
    _, resp = http_get_json(url, headers)
    return resp

def print_status_table(tasks: list[dict], elapsed: float):
    """打印所有任务的状态表格。"""
    # 移到行首并清屏（每次刷新覆盖）
    print(f"\r\033[{len(tasks)+4}A", end="")  # 上移 N 行

    print(colorize(f"\n{'═'*72}", "bold"))
    print(colorize(f"  图像编辑任务并发测试   已用时: {elapsed:.0f}s", "bold"))
    print(colorize(f"{'═'*72}", "bold"))

    terminal_statuses = {"completed", "failed", "submit_failed"}
    done_count = sum(1 for t in tasks if t["status"] in terminal_statuses)

    for t in tasks:
        idx = t["idx"]
        task_id = t.get("id", "—")
        status = t["status"]
        error = t.get("error") or ""
        result_count = len(t.get("result_urls", []))

        if status == "pending":
            status_str = colorize("⏳ pending    ", "pending")
        elif status == "processing":
            status_str = colorize("⚙  processing ", "processing")
        elif status == "completed":
            status_str = colorize(f"✓  completed  [{result_count} 张]", "completed")
        elif status in ("failed", "submit_failed"):
            status_str = colorize(f"✗  {status:<12}", "failed")
        else:
            status_str = colorize(f"?  {status:<12}", "dim")

        id_str = str(task_id).rjust(6) if task_id != "—" else "  —   "
        err_str = f"  {colorize(error[:40], 'failed')}" if error else ""
        print(f"  #{idx+1:02d}  id={id_str}  {status_str}{err_str}")

    print(colorize(f"{'─'*72}", "dim"))
    print(f"  完成: {done_count}/{len(tasks)}   按 Ctrl+C 退出轮询\n")

def main():
    parser = argparse.ArgumentParser(description="并发测试 /v1/images/tasks/edits")
    parser.add_argument("--url",           default="http://localhost:9090", help="网关基础 URL")
    parser.add_argument("--key",           required=True,                   help="API Key (Bearer token)")
    parser.add_argument("--model",         required=True,                   help="模型名称, 例如 gpt-image-2")
    parser.add_argument("--image",         required=True,                   help="产品图片路径 (jpg/png)")
    parser.add_argument("--concurrency",   type=int, default=10,            help="并发任务数 (默认10)")
    parser.add_argument("--n",             type=int, default=1,             help="每个任务生成张数 (1-4)")
    parser.add_argument("--size",          default="1024x1024",             help="图片尺寸")
    parser.add_argument("--poll-interval", type=float, default=5.0,         help="轮询间隔秒数 (默认5)")
    parser.add_argument("--timeout",       type=int, default=300,           help="最大等待秒数 (默认300)")
    args = parser.parse_args()

    # 读取并 base64 编码图片
    img_path = Path(args.image)
    if not img_path.exists():
        print(f"错误: 图片文件不存在: {img_path}", file=sys.stderr)
        sys.exit(1)

    with open(img_path, "rb") as f:
        image_b64 = base64.b64encode(f.read()).decode("utf-8")
    print(f"图片已加载: {img_path.name} ({img_path.stat().st_size/1024:.1f} KB, base64: {len(image_b64)/1024:.1f} KB)")

    headers = {
        "Authorization": f"Bearer {args.key}",
        "Content-Type": "application/json",
    }

    # ── 阶段1：并发提交10个任务 ──────────────────────────────────────────────
    print(f"\n正在并发提交 {args.concurrency} 个任务 → {args.url}/v1/images/tasks/edits ...")
    tasks: list[dict] = [None] * args.concurrency

    def _submit(idx):
        return submit_task(idx, args.url, headers, image_b64, args.model, args.size, args.n)

    with ThreadPoolExecutor(max_workers=args.concurrency) as pool:
        futures = {pool.submit(_submit, i): i for i in range(args.concurrency)}
        for fut in as_completed(futures):
            result = fut.result()
            tasks[result["idx"]] = result
            status_icon = "✓" if result["id"] else "✗"
            print(f"  {status_icon} #{result['idx']+1:02d} 提交{'成功' if result['id'] else '失败'}: "
                  f"id={result.get('id', '—')}  status={result['status']}"
                  + (f"  错误: {result['error']}" if result['error'] else ""))

    submitted = [t for t in tasks if t["id"] is not None]
    print(f"\n提交完成: {len(submitted)}/{args.concurrency} 个任务成功入队。")

    if not submitted:
        print("所有任务提交失败，退出。")
        sys.exit(1)

    # ── 阶段2：轮询任务状态 ───────────────────────────────────────────────────
    terminal_statuses = {"completed", "failed", "submit_failed"}
    start_time = time.time()

    # 打印初始表格（占位行）
    print()
    for _ in range(len(tasks) + 4):
        print()

    try:
        while True:
            elapsed = time.time() - start_time

            # 并发轮询所有未完成任务
            def _poll(t):
                if t["id"] is None or t["status"] in terminal_statuses:
                    return t
                data = poll_task(t["id"], args.url, headers)
                t["status"] = data.get("status", t["status"])
                t["result_urls"] = data.get("result_urls") or []
                t["error"] = data.get("error_message") or t.get("error")
                return t

            with ThreadPoolExecutor(max_workers=min(len(tasks), 10)) as pool:
                updated = list(pool.map(_poll, tasks))
            tasks = updated

            print_status_table(tasks, elapsed)

            all_done = all(t["status"] in terminal_statuses for t in tasks)
            if all_done:
                print(colorize("所有任务已完成！", "completed"))
                break

            if elapsed > args.timeout:
                print(colorize(f"超时（{args.timeout}s），停止轮询。", "failed"))
                break

            time.sleep(args.poll_interval)

    except KeyboardInterrupt:
        print("\n\n用户中断轮询。")

    # ── 最终汇总 ──────────────────────────────────────────────────────────────
    print("\n" + colorize("═"*72, "bold"))
    print(colorize("  最终结果", "bold"))
    print(colorize("═"*72, "bold"))
    for t in tasks:
        if t["status"] == "completed":
            urls = t.get("result_urls") or []
            print(f"  #{t['idx']+1:02d} id={t['id']}  {colorize('completed', 'completed')}  {len(urls)} 张结果")
            for url in urls:
                print(f"       {colorize(url, 'dim')}")
        elif t["id"]:
            print(f"  #{t['idx']+1:02d} id={t['id']}  {colorize(t['status'], 'failed')}  {t.get('error','')}")
        else:
            print(f"  #{t['idx']+1:02d} 提交失败  {colorize(t.get('error',''), 'failed')}")

    completed = sum(1 for t in tasks if t["status"] == "completed")
    failed    = sum(1 for t in tasks if t["status"] in ("failed", "submit_failed"))
    pending   = sum(1 for t in tasks if t["status"] not in terminal_statuses)
    total_elapsed = time.time() - start_time
    print(colorize(f"\n  完成:{completed}  失败:{failed}  未完成:{pending}  总用时:{total_elapsed:.0f}s", "bold"))

if __name__ == "__main__":
    main()
