import { parseSSEStream } from "./stream";

it("parses frames split across chunks and ignores comments", async () => {
  const enc = new TextEncoder();
  const chunks = [
    ': ping\n\ndata: {"type":"start","timestamp":"t","message":"go","data":{}}\n',
    '\ndata: {"type":"complete","timestamp":"t","data":{"total_scenarios":1}}\n\n',
  ];
  const body = new ReadableStream<Uint8Array>({
    start(c) {
      chunks.forEach((chunk) => c.enqueue(enc.encode(chunk)));
      c.close();
    },
  });
  const seen: string[] = [];
  await parseSSEStream(body, (e) => seen.push(e.type));
  expect(seen).toEqual(["start", "complete"]);
});
