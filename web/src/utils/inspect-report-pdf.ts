import html2pdf from "html2pdf.js";
import { http } from "../services/http";

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
    "position:fixed;left:-10000px;top:0;width:1100px;height:100%;border:0;visibility:hidden;";
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

async function waitForRender(doc: Document) {
  try {
    await doc.fonts?.ready;
  } catch {
    // ignore
  }
  const imgs = Array.from(doc.images);
  await Promise.all(
    imgs.map(
      (img) =>
        new Promise<void>((resolve) => {
          if (img.complete) resolve();
          else {
            img.onload = () => resolve();
            img.onerror = () => resolve();
          }
        }),
    ),
  );
  await new Promise((r) => window.setTimeout(r, 320));
}

/**
 * 在浏览器端用 html2pdf.js（html2canvas + jsPDF）将巡检 HTML 转为 PDF。
 * 不依赖服务端 wkhtmltopdf / Chrome，样式与 HTML 预览一致。
 */
export async function downloadInspectReportPdf(apiHtmlPath: string, filename: string) {
  const html = await fetchReportHtml(apiHtmlPath);
  const { root, cleanup } = mountHtmlInIframe(html);
  try {
    const doc = root.ownerDocument;
    await waitForRender(doc);
    const width = Math.max(root.scrollWidth, root.clientWidth, 960);
    await html2pdf()
      .set({
        margin: [6, 6, 8, 6],
        filename,
        image: { type: "jpeg", quality: 0.98 },
        html2canvas: {
          scale: 2,
          useCORS: true,
          logging: false,
          backgroundColor: "#f1f5f9",
          windowWidth: width,
          width,
        },
        jsPDF: { unit: "mm", format: "a4", orientation: "portrait" },
        pagebreak: { mode: ["css", "legacy"] },
      })
      .from(root)
      .save();
  } finally {
    cleanup();
  }
}
