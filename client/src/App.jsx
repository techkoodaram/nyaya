import { useState } from "react";

const API_URL = import.meta.env.VITE_API_URL || "/api/retrieve";

function App() {
  const [inputText, setInputText] = useState("");
  const [responseData, setResponseData] = useState(null);
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  const handleSend = async () => {
    const trimmed = inputText.trim();
    if (!trimmed) {
      setError("Please enter text before sending.");
      return;
    }

    setIsLoading(true);
    setError("");

    try {
      const response = await fetch(API_URL, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ query: trimmed, topK: 3 }),
      });

      if (!response.ok) {
        const errBody = await response.json().catch(() => ({}));
        throw new Error(
          errBody.error || `Request failed with status ${response.status}`
        );
      }

      const data = await response.json();
      setResponseData(data);
    } catch (err) {
      setResponseData(null);
      setError(err.message || "Failed to retrieve data.");
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <main className="page">
      <div className="bgOrb orbOne" />
      <div className="bgOrb orbTwo" />

      <section className="card">
        <p className="eyebrow">Nyaya Assistant</p>
        <h1>Ask Anything About Indian Law</h1>
        <p className="subtitle">
          Enter your legal query and retrieve contextual answers from your backend
          service.
        </p>

        <div className="controls">
          <textarea
            className="input"
            rows={4}
            placeholder="Type your query..."
            value={inputText}
            onChange={(e) => setInputText(e.target.value)}
            disabled={isLoading}
          />
          <button className="sendButton" onClick={handleSend} disabled={isLoading}>
            {isLoading ? "Sending..." : "Send"}
          </button>
        </div>

        {error && <p className="error">{error}</p>}

        <section className="resultSection">
          <div className="resultHeader">
            <h2>Response</h2>
            <span className="resultPill">{responseData ? "Updated" : "Awaiting input"}</span>
          </div>
          {responseData ? (
            <>
              <div className="answerBlock">
                <h3>Answer</h3>
                <p>{responseData.answer || "No answer generated."}</p>
              </div>

              <div className="refsBlock">
                <h3>References ({responseData.retrievedCount || 0})</h3>
                {Array.isArray(responseData.references) &&
                responseData.references.length > 0 ? (
                  <ul className="refsList">
                    {responseData.references.map((ref) => (
                      <li className="refItem" key={ref.id}>
                        <div className="refTop">
                          <strong>{ref.title}</strong>
                          <span>
                            {String(ref.source).toUpperCase()} | score {ref.score}
                          </span>
                        </div>
                        <p>{ref.excerpt}</p>
                      </li>
                    ))}
                  </ul>
                ) : (
                  <p className="empty">No references found.</p>
                )}
              </div>
            </>
          ) : (
            <p className="empty">No response yet.</p>
          )}
        </section>
      </section>
    </main>
  );
}

export default App;
