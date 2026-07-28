import { gzipSync } from "node:zlib";
import { readdir, readFile, writeFile } from "node:fs/promises";
import { extname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("../dist/", import.meta.url));
const compressible = new Set([
  ".css",
  ".html",
  ".js",
  ".json",
  ".svg",
  ".txt",
  ".webmanifest",
  ".xml",
]);
const minimumBytes = 1024;

async function visit(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  await Promise.all(
    entries.map(async (entry) => {
      const path = join(directory, entry.name);
      if (entry.isDirectory()) {
        await visit(path);
        return;
      }
      if (!entry.isFile() || !compressible.has(extname(entry.name))) {
        return;
      }
      const source = await readFile(path);
      if (source.byteLength < minimumBytes) {
        return;
      }
      const compressed = gzipSync(source, { level: 9 });
      if (compressed.byteLength < source.byteLength) {
        await writeFile(`${path}.gz`, compressed, { mode: 0o644 });
      }
    }),
  );
}

await visit(root);
