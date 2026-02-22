import { createFileRoute, Link } from "@tanstack/react-router";
import { useState } from "react";

export const Route = createFileRoute("/mcp_client")({
  component: RouteComponent,
});

function RouteComponent() {
  const mcpClientUrl = import.meta.env.VITE_MCP_CLIENT_URL;
  const [prompt, setPrompt] = useState<string>('');
  const [currentMessage, setCurrentMessage] = useState<string>("");
  const [history, setHistory] = useState<string[]>([]);

  const send = async () => {
    setCurrentMessage("");

    const res = await fetch(`${mcpClientUrl}/mcp_client/chat`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ prompt }),
    });

    const reader = res.body?.getReader();
    const decoder = new TextDecoder();

    let accumulatedText = "";

    while (reader) {
      const { done, value } = await reader.read();
      if (done) break;

      const chunk = decoder.decode(value, { stream: true });
      const cleanChunk = chunk.replaceAll("```", "`");

      accumulatedText += cleanChunk;
      setCurrentMessage(accumulatedText);
    }

    setHistory((prev) => [...prev, accumulatedText]);
    setCurrentMessage("");
  };

  return (
    <div>
      <Link to="/">home</Link>
      <div>
        <input className="border mr-2" type="text" value={prompt || ""} onChange={(e) => setPrompt(e.target.value)} />
        <button onClick={send} className="border hover:cursor-pointer">
          Send
        </button>
      </div>
      <div>
        <div className="chat-container">
          {history.map((msg, i) => (
            <div key={i} className="message old">
              {msg}
            </div>
          ))}

          {currentMessage && (
            <div className="message streaming">
              {currentMessage}
              <span className="cursor">|</span>{" "}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
