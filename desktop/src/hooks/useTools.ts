import { useState, useEffect, useCallback } from "react";
import type { ToolStatus, ScanError } from "../lib/types";
import { api, ConfigResult } from "../lib/tauri";

export function useTools() {
  const [tools, setTools] = useState<ToolStatus[]>([]);
  const [scanErrors, setScanErrors] = useState<ScanError[]>([]);
  const [scanning, setScanning] = useState(false);
  const [configuring, setConfiguring] = useState(false);

  const scan = useCallback(async () => {
    setScanning(true);
    try {
      const result = await api.scanTools();
      setTools(result.tools);
      setScanErrors(result.scan_errors);
    } catch (err) {
      console.error("扫描失败:", err);
    } finally {
      setScanning(false);
    }
  }, []);

  useEffect(() => { scan(); }, [scan]);

  const configureTool = useCallback(async (tool: string, model?: string): Promise<ConfigResult> => {
    setConfiguring(true);
    try {
      const result = await api.configureTool(tool, model);
      await scan();
      return result;
    } finally {
      setConfiguring(false);
    }
  }, [scan]);

  const clearTool = useCallback(async (tool: string): Promise<void> => {
    await api.clearToolConfig(tool);
    await scan();
  }, [scan]);

  const configureAll = useCallback(async (forceAll = false): Promise<ConfigResult[]> => {
    setConfiguring(true);
    try {
      const targets = forceAll ? tools : tools.filter(t => !t.configured);
      if (targets.length === 0) return [];
      const results = await api.configureTools(targets.map(t => t.tool));
      await scan();
      return results;
    } finally {
      setConfiguring(false);
    }
  }, [tools, scan]);

  return { tools, scanErrors, scanning, configuring, scan, configureAll, configureTool, clearTool };
}
