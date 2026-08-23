import { http } from "../services/http";
import { downloadElementAsPdf } from "./html-to-pdf";

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

function mountHtmlInIframe(html: string): { cleanup: () => void; root: HTMLElement; doc: Document } {
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
    doc,
    cleanup: () => {
      if (iframe.parentNode) iframe.parentNode.removeChild(iframe);
    },
  };
}

/**
 * 参考 PromAI：html2canvas + jsPDF 将巡检 HTML 转为 PDF，样式与 HTML 预览一致。
 */
export async function downloadInspectReportPdf(apiHtmlPath: string, filename: string) {
  const html = await fetchReportHtml(apiHtmlPath);
  const { root, cleanup } = mountHtmlInIframe(html);
  try {
    await downloadElementAsPdf({ filename, target: root, orientation: "landscape", scale: 1.5 });
  } finally {
    cleanup();
  }
}
