import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import LotteryPage from "./LotteryPage";

const apiMocks = vi.hoisted(() => ({
  useLotteryCurrentEvent: vi.fn(),
  useLotteryWinnerRecords: vi.fn(),
  useUserLotteryRecords: vi.fn(),
}));

vi.mock("@/hooks/use-api", () => apiMocks);

describe("LotteryPage winner records", () => {
  beforeEach(() => {
    apiMocks.useLotteryCurrentEvent.mockReturnValue({
      data: { event: null, prizes: [] },
      isLoading: false,
    });
    apiMocks.useLotteryWinnerRecords.mockReturnValue({
      data: {
        records: [{
          id: 9,
          masked_phone: "138****5678",
          prize_name: "等额充值",
          prize_type: "match_recharge",
          prize_value: 200,
          created_at: "2026-07-27T08:30:00Z",
        }],
        total: 1,
      },
      isLoading: false,
    });
    apiMocks.useUserLotteryRecords.mockReturnValue({
      data: {
        records: [{
          id: 9,
          prize_name: "等额充值",
          prize_type: "match_recharge",
          prize_value: 200,
          recharge_amount: 200,
          created_at: "2026-07-27T08:30:00Z",
        }],
        total: 1,
      },
      isLoading: false,
    });
  });

  it("renders public winner columns without recharge amount", () => {
    const html = renderToStaticMarkup(<LotteryPage />);

    expect(html).toContain("中奖记录");
    expect(html).toContain("中奖用户");
    expect(html).toContain("138****5678");
    expect(html).not.toContain("我的抽奖记录");
    expect(html).not.toContain("充值金额");
  });
});
