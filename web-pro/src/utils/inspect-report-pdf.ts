// @ts-nocheck
import { http } from "../services/http";
import { renderElementAsPdfBlob } from "./html-to-pdf";

async function fetchReportHtml(apiPath: string): Promise<string> {
  const raw: unknown = await http.get(apiPath, { responseType: "text", timeout: 120_000 });
  if (typeof raw === "string") return raw;
  if (raw instanceof Blob) return raw.text();
  if (raw && typeof raw === "object" && "data" in raw) {
    const inner = (raw as { data: unknown }).data;
    if (typeof inner === "string") return inner;
    if (inner instanceof Blob) return inner.text();
  }
  throw new Error("无法读取 HTML 报告内容");
}

/**
 * 离屏 iframe 的渲染宽度（px）。
 * 报告模板按 960px 主栏设计，A4 竖版按此宽度缩放后正文约 10.5pt，
 * 用 1200px 渲染再压到 194mm 会让字号偏小、表格更容易挤在一起。
 */
const RENDER_WIDTH_PX = 960;

function mountHtmlInIframe(html: string): { cleanup: () => void; root: HTMLElement } {
  const iframe = document.createElement("iframe");
  iframe.setAttribute("aria-hidden", "true");
  iframe.style.cssText =
    `position:fixed;left:0;top:0;width:${RENDER_WIDTH_PX}px;height:100%;border:0;z-index:-1;opacity:0;pointer-events:none;`;
  document.body.appendChild(iframe);
  const doc = iframe.contentDocument;
  if (!doc) {
    document.body.removeChild(iframe);
    throw new Error("无法创建 PDF 导出容器");
  }
  doc.open();
  doc.write(html);
  doc.close();
  const root = (doc.querySelector(".page") as HTMLElement | null) ?? doc.body;
  return {
    root,
    cleanup: () => {
      if (iframe.parentNode) iframe.parentNode.removeChild(iframe);
    },
  };
}

function downloadBlob(blob: Blob, filename: string) {
  const a = document.createElement("a");
  a.href = URL.createObjectURL(blob);
  a.download = filename;
  a.click();
  setTimeout(() => URL.revokeObjectURL(a.href), 5000);
}

export async function uploadInspectReportPdf(projectId: number, runId: number, blob: Blob, filename: string) {
  const fd = new FormData();
  fd.append("pdf", blob, filename);
  await http.post(`/projects/${projectId}/inspect/runs/${runId}/report.pdf`, fd, {
    headers: { "Content-Type": "multipart/form-data" },
    timeout: 120_000,
  });
}

/**
 * html2canvas + jsPDF 生成 PDF，下载并上传到服务端（邮件/API 复用同一份）。
 */
export async function downloadInspectReportPdf(
  projectId: number,
  apiHtmlPath: string,
  filename: string,
) {
  const html = await fetchReportHtml(apiHtmlPath);
  const { root, cleanup } = mountHtmlInIframe(html);
  try {
    // 竖版 + scale 2：巡检报告是纵向长文档，横版会让每页只剩两三行表格；
    // scale 2 让 300 dpi 打印下的中文不发虚。
    const blob = await renderElementAsPdfBlob(root, "portrait", 2);
    downloadBlob(blob, filename);

    const runMatch = apiHtmlPath.match(/\/runs\/(\d+)\//);
    const runId = runMatch ? Number(runMatch[1]) : 0;
    if (projectId > 0 && runId > 0) {
      await uploadInspectReportPdf(projectId, runId, blob, filename).catch(() => undefined);
    }
  } finally {
    cleanup();
  }
}
