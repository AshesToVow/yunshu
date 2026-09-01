// @ts-nocheck
import html2canvas from "html2canvas";
import { jsPDF } from "jspdf";

export type HtmlToPdfOptions = {
  filename: string;
  target: HTMLElement;
  orientation?: "portrait" | "landscape";
  scale?: number;
};

/** A4 尺寸（mm）。竖版贴合巡检报告的窄栏排版，横版留给宽表格。 */
const A4 = { portrait: { w: 210, h: 297 }, landscape: { w: 297, h: 210 } } as const;

/** 页边距（mm），避免打印/装订吃掉内容。 */
const PAGE_MARGIN_MM = 8;

/**
 * 分页优先在这些元素的下边界断开，避免把卡片、表格行、章节标题切成两半。
 * 取「所有候选断点中离理想页底最近且不超过它」的那个。
 */
const BREAK_CANDIDATE_SELECTOR = [
  "section",
  ".card",
  ".kpi",
  ".block",
  "table",
  "thead",
  "tr",
  "h1",
  "h2",
  "h3",
  "p",
  "li",
].join(",");

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

/**
 * 收集可安全分页的纵坐标（相对 target 顶部，CSS px，升序去重）。
 * html2canvas 产出的是一整张长图，没有断点信息就只能等分切割，
 * 那样必然出现「同一行文字上下两页各一半」的观感问题。
 */
function collectBreakOffsets(target: HTMLElement): number[] {
  const baseTop = target.getBoundingClientRect().top;
  const offsets = new Set<number>();
  target.querySelectorAll<HTMLElement>(BREAK_CANDIDATE_SELECTOR).forEach((el) => {
    const rect = el.getBoundingClientRect();
    if (rect.height <= 0) return;
    const bottom = rect.bottom - baseTop;
    if (bottom > 0) offsets.add(Math.round(bottom));
  });
  return Array.from(offsets).sort((a, b) => a - b);
}

/**
 * 在 [minCut, idealCut] 内挑最靠下的安全断点。
 * 找不到（例如单个元素本身高于一页）时返回 idealCut 硬切，保证推进。
 */
function pickCut(breakOffsets: number[], idealCut: number, minCut: number): number {
  let best = -1;
  for (const offset of breakOffsets) {
    if (offset > idealCut) break;
    if (offset >= minCut) best = offset;
  }
  return best > 0 ? best : idealCut;
}

async function captureCanvas(target: HTMLElement, scale: number) {
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
    return { canvas, cssHeight: height };
  } finally {
    doc.body.style.cssText = originalBodyStyle;
    doc.documentElement.style.cssText = originalHtmlStyle;
  }
}

/**
 * 按安全断点把长图切成多页 PDF。
 *
 * 与等分切割的区别：每页高度不固定，但缩放比例始终一致（宽度铺满内容区），
 * 因此各页字号相同，且不会在表格行/卡片中间断开。
 */
function canvasToPdfBlob(
  canvas: HTMLCanvasElement,
  orientation: "portrait" | "landscape",
  cssHeight: number,
  breakOffsets: number[],
): Blob {
  const { w: pageW, h: pageH } = A4[orientation];
  const pdf = new jsPDF({ orientation, unit: "mm", format: "a4" });
  const contentW = pageW - PAGE_MARGIN_MM * 2;
  const contentH = pageH - PAGE_MARGIN_MM * 2;

  const pxPerMm = canvas.width / contentW;
  const canvasPerCss = canvas.height / Math.max(cssHeight, 1);
  const idealPageCss = (contentH * pxPerMm) / canvasPerCss;
  // 一页至少装下 35% 内容，避免为对齐断点产生大片空白。
  const minPageCss = idealPageCss * 0.35;

  let cursorCss = 0;
  let firstPage = true;
  let guard = 0;

  while (cursorCss < cssHeight - 1 && guard++ < 500) {
    const idealCut = Math.min(cursorCss + idealPageCss, cssHeight);
    const cutCss =
      idealCut >= cssHeight ? cssHeight : pickCut(breakOffsets, idealCut, cursorCss + minPageCss);

    const sliceTop = Math.round(cursorCss * canvasPerCss);
    const sliceHeight = Math.min(
      Math.round((cutCss - cursorCss) * canvasPerCss),
      canvas.height - sliceTop,
    );
    if (sliceHeight <= 0) break;

    const slice = document.createElement("canvas");
    slice.width = canvas.width;
    slice.height = sliceHeight;
    const ctx = slice.getContext("2d");
    if (!ctx) throw new Error("无法创建 PDF 画布");
    // 切片默认透明，转 JPEG 会变黑，先铺白底。
    ctx.fillStyle = "#ffffff";
    ctx.fillRect(0, 0, slice.width, slice.height);
    ctx.drawImage(canvas, 0, sliceTop, canvas.width, sliceHeight, 0, 0, canvas.width, sliceHeight);

    if (!firstPage) pdf.addPage();
    firstPage = false;
    pdf.addImage(
      slice.toDataURL("image/jpeg", 0.92),
      "JPEG",
      PAGE_MARGIN_MM,
      PAGE_MARGIN_MM,
      contentW,
      Math.min(sliceHeight / pxPerMm, contentH),
    );

    cursorCss = cutCss;
  }

  return pdf.output("blob");
}

/** html2canvas 截图 + jsPDF 智能分页，返回 PDF Blob。 */
export async function renderElementAsPdfBlob(
  target: HTMLElement,
  orientation: "portrait" | "landscape" = "portrait",
  scale = 2,
): Promise<Blob> {
  const breakOffsets = collectBreakOffsets(target);
  const { canvas, cssHeight } = await captureCanvas(target, scale);
  return canvasToPdfBlob(canvas, orientation, cssHeight, breakOffsets);
}

export async function downloadElementAsPdf({
  filename,
  target,
  orientation = "portrait",
  scale = 2,
}: HtmlToPdfOptions) {

  const blob = await renderElementAsPdfBlob(target, orientation, scale);
  const a = document.createElement("a");
  a.href = URL.createObjectURL(blob);
  a.download = filename;
  a.click();
  setTimeout(() => URL.revokeObjectURL(a.href), 5000);
}
