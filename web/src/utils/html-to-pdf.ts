import html2canvas from "html2canvas";
import { jsPDF } from "jspdf";

export type HtmlToPdfOptions = {
  filename: string;
  target: HTMLElement;
  orientation?: "portrait" | "landscape";
  scale?: number;
};

async function waitForPageReady(doc: Document) {
  if (doc.readyState !== "complete") {
    await new Promise<void>((resolve) => {
      if (doc.readyState === "complete") resolve();
      else doc.defaultView?.addEventListener("load", () => resolve(), { once: true });
    });
  }
  try {
    await doc.fonts?.ready;
  } catch {
    // ignore
  }
  await Promise.all(
    Array.from(doc.images).map(
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
  await new Promise((r) => window.setTimeout(r, 600));
  await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()));
  await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()));
}

function applyCloneStyles(clonedDoc: Document) {
  const clonedBody = clonedDoc.body;
  const clonedHtml = clonedDoc.documentElement;
  if (clonedBody) {
    clonedBody.style.visibility = "visible";
    clonedBody.style.opacity = "1";
    clonedBody.style.backgroundColor = "#ffffff";
    clonedBody.style.margin = "0";
    clonedBody.style.padding = "0";
  }
  if (clonedHtml) {
    clonedHtml.style.visibility = "visible";
    clonedHtml.style.opacity = "1";
    clonedHtml.style.backgroundColor = "#ffffff";
    clonedHtml.style.margin = "0";
    clonedHtml.style.padding = "0";
  }
  const page = clonedDoc.querySelector(".page") as HTMLElement | null;
  if (page) {
    page.style.backgroundColor = "#ffffff";
    page.style.maxWidth = "none";
    page.style.width = `${page.scrollWidth || 960}px`;
  }
  clonedDoc.querySelectorAll("img").forEach((img) => {
    img.style.visibility = "visible";
    img.style.opacity = "1";
  });
}

async function captureCanvas(target: HTMLElement, orientation: "portrait" | "landscape", scale: number) {
  const doc = target.ownerDocument;
  await waitForPageReady(doc);

  const originalBodyStyle = doc.body.style.cssText;
  const originalHtmlStyle = doc.documentElement.style.cssText;
  doc.body.style.visibility = "visible";
  doc.body.style.opacity = "1";
  doc.body.style.backgroundColor = "#ffffff";
  doc.documentElement.style.backgroundColor = "#ffffff";

  const width = Math.max(target.scrollWidth, target.clientWidth, 960);
  const height = Math.max(target.scrollHeight, target.clientHeight, 400);

  try {
    const canvas = await html2canvas(target, {
      scale,
      useCORS: true,
      logging: false,
      backgroundColor: "#ffffff",
      width,
      height,
      windowWidth: width,
      windowHeight: height,
      scrollX: 0,
      scrollY: 0,
      foreignObjectRendering: false,
      imageTimeout: 30_000,
      onclone: (clonedDoc) => applyCloneStyles(clonedDoc),
    });
    if (!canvas.width || !canvas.height) {
      throw new Error("页面截图失败：内容尺寸为 0");
    }
    return canvas;
  } finally {
    doc.body.style.cssText = originalBodyStyle;
    doc.documentElement.style.cssText = originalHtmlStyle;
  }
}

function canvasToPdfBlob(canvas: HTMLCanvasElement, orientation: "portrait" | "landscape"): Blob {
  const pageW = orientation === "landscape" ? 297 : 210;
  const pageH = orientation === "landscape" ? 210 : 297;
  const pdf = new jsPDF({ orientation, unit: "mm", format: "a4" });
  const imgWidth = pageW;
  const imgHeight = (canvas.height * imgWidth) / canvas.width;
  const totalPages = Math.max(1, Math.ceil(imgHeight / pageH));

  for (let page = 0; page < totalPages; page++) {
    if (page > 0) pdf.addPage();
    const sourceY = (canvas.height / totalPages) * page;
    const sourceHeight = Math.min(canvas.height / totalPages, canvas.height - sourceY);

    const tempCanvas = document.createElement("canvas");
    tempCanvas.width = canvas.width;
    tempCanvas.height = sourceHeight;
    const tempCtx = tempCanvas.getContext("2d");
    if (!tempCtx) throw new Error("无法创建 PDF 画布");
    tempCtx.drawImage(canvas, 0, sourceY, canvas.width, sourceHeight, 0, 0, canvas.width, sourceHeight);

    const imgData = tempCanvas.toDataURL("image/jpeg", 0.92);
    const pageImgHeight = (sourceHeight * imgWidth) / canvas.width;
    pdf.addImage(imgData, "JPEG", 0, 0, imgWidth, pageImgHeight);
  }

  return pdf.output("blob");
}

/** PromAI 同款：html2canvas 截图 + jsPDF 分页，返回 PDF Blob。 */
export async function renderElementAsPdfBlob(
  target: HTMLElement,
  orientation: "portrait" | "landscape" = "landscape",
  scale = 1.5,
): Promise<Blob> {
  const canvas = await captureCanvas(target, orientation, scale);
  return canvasToPdfBlob(canvas, orientation);
}

export async function downloadElementAsPdf({
  filename,
  target,
  orientation = "landscape",
  scale = 1.5,
}: HtmlToPdfOptions) {
  const blob = await renderElementAsPdfBlob(target, orientation, scale);
  const a = document.createElement("a");
  a.href = URL.createObjectURL(blob);
  a.download = filename;
  a.click();
  setTimeout(() => URL.revokeObjectURL(a.href), 5000);
}
