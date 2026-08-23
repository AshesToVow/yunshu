/// <reference types="vite/client" />

declare module "html2pdf.js" {
  interface Html2PdfWorker {
    set(options: Record<string, unknown>): Html2PdfWorker;
    from(element: HTMLElement): Html2PdfWorker;
    save(): Promise<void>;
  }
  function html2pdf(): Html2PdfWorker;
  export default html2pdf;
}