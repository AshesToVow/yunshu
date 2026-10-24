/** Fill apiVersion / kind / namespace into a CR template YAML body. */
export function materializeCrTemplateBody(
  body: string,
  opts: {
    namespace?: string;
    apiVersion?: string;
    kind?: string;
  },
): string {
  let out = body.trim();
  if (opts.apiVersion) {
    out = out.replace(/^apiVersion:\s*.*$/m, `apiVersion: ${opts.apiVersion}`);
  }
  if (opts.kind) {
    out = out.replace(/^kind:\s*.*$/m, `kind: ${opts.kind}`);
  }
  if (opts.namespace) {
    if (/^\s*namespace:\s*.+$/m.test(out)) {
      out = out.replace(/^\s*namespace:\s*.*$/m, `  namespace: ${opts.namespace}`);
    } else if (/^metadata:\s*$/m.test(out)) {
      out = out.replace(/^metadata:\s*$/m, `metadata:\n  namespace: ${opts.namespace}`);
    } else if (/^metadata:\s*\n/m.test(out)) {
      out = out.replace(/^(metadata:\s*\n)/m, `$1  namespace: ${opts.namespace}\n`);
    }
  }
  return out;
}
