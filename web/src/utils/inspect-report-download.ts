// 巡检报告下载/打开入口（RF-10）。
// 由 web/src/pages/project-inspect-page.tsx 原样搬迁，与 inspect-report-pdf.ts / html-to-pdf.ts 同层，
// 统一收口「带鉴权的 http 取 blob → 本地下载/新窗口打开」这条链路，避免下载逻辑散落在页面里。
import { message } from "antd";
import { extractApiErrorMessage, http } from "../services/http";
import { checkInspectReportPdf, inspectReportHtmlUrl, inspectReportPdfUrl } from "../services/inspect";

/** 兼容 http 拦截器可能返回 Blob / {data} / 字符串 / ArrayBuffer 的几种形态 */
export function toReportBlob(raw: unknown, type: string): Blob {
  if (raw instanceof Blob) return raw;
  if (raw && typeof raw === "object" && "data" in raw) {
    const inner = (raw as { data: unknown }).data;
    if (inner instanceof Blob) return inner;
    if (typeof inner === "string" || inner instanceof ArrayBuffer || ArrayBuffer.isView(inner)) {
      return new Blob([inner as BlobPart], { type });
    }
  }
  if (typeof raw === "string" || raw instanceof ArrayBuffer || ArrayBuffer.isView(raw)) {
    return new Blob([raw as BlobPart], { type });
  }
  return new Blob([], { type });
}

/** 走带鉴权的 http 拉取报告后再以 blob URL 打开新窗口（直接 window.open 会丢 Authorization 头） */
export function openAuthorized(url: string) {
  void http
    .get(url, { responseType: "blob" })
    .then((raw: unknown) => {
      const type = url.endsWith(".pdf")
        ? "application/pdf"
        : url.endsWith(".xlsx")
          ? "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
          : "text/html;charset=utf-8";
      const blob = toReportBlob(raw, type);
      const obj = URL.createObjectURL(blob);
      window.open(obj, "_blank");
      setTimeout(() => URL.revokeObjectURL(obj), 60_000);
    })
    .catch((e) => message.error(extractApiErrorMessage(e, "打开报告失败")));
}

export function downloadBlobFile(blob: Blob, filename: string) {
  const a = document.createElement("a");
  a.href = URL.createObjectURL(blob);
  a.download = filename;
  a.click();
  setTimeout(() => URL.revokeObjectURL(a.href), 5000);
}

/** 优先下载后端已生成的 PDF，不存在时回落到前端 HTML → PDF 渲染 */
export function downloadInspectPdf(projectId: number, runId: number, projectName?: string) {
  const key = "inspect-pdf";
  message.loading({ content: "正在准备 PDF…", key, duration: 0 });
  void (async () => {
    try {
      const st = await checkInspectReportPdf(projectId, runId);
      const filename =
        (st.filename && st.filename.trim()) ||
        buildInspectReportFilename(projectName, runId, "pdf");
      if (st.exists) {
        const raw: unknown = await http.get(inspectReportPdfUrl(projectId, runId), {
          responseType: "blob",
          timeout: 120_000,
        });
        downloadBlobFile(toReportBlob(raw, "application/pdf"), filename);
        message.success({ content: "PDF 已下载", key });
        return;
      }
      const { downloadInspectReportPdf } = await import("./inspect-report-pdf");
      await downloadInspectReportPdf(projectId, inspectReportHtmlUrl(projectId, runId), filename);
      message.success({ content: "PDF 已生成并保存（与 HTML 样式一致）", key });
    } catch (e) {
      message.error({ content: extractApiErrorMessage(e, "生成 PDF 失败"), key });
    }
  })();
}

/** 生成可读报告文件名：项目名-巡检-{runId}.ext */
export function buildInspectReportFilename(projectName: string | undefined, runId: number, ext: string) {
  const safeExt = (ext || "bin").replace(/^\./, "");
  let base = (projectName || "").trim().replace(/[\\/:*?"<>|]/g, "_").replace(/[.\s]+$/g, "");
  if (!base) {
    base = `inspect-run-${runId}`;
  } else if (runId > 0) {
    base = `${base}-巡检-${runId}`;
  }
  return `${base}.${safeExt}`;
}
