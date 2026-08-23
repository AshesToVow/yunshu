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

function mountHtmlInIframe(html: string): { cleanup: () => void; root: HTMLElement } {
  const iframe = document.createElement("iframe");
  iframe.setAttribute("aria-hidden", "true");
  iframe.style.cssText =
    "position:fixed;left:0;top:0;width:1200px;height:100%;border:0;z-index:-1;opacity:0;pointer-events:none;";
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
    const blob = await renderElementAsPdfBlob(root, "landscape", 1.5);
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
