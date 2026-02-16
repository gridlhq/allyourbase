import { useState, useCallback, useRef } from "react";
import { executeSQL, ApiError } from "../api";
import type { SqlResult } from "../types";
import { Play, AlertCircle, Clock } from "lucide-react";

export function SqlEditor() {
  const [query, setQuery] = useState(
    () => localStorage.getItem("ayb_sql_query") || "SELECT 1 AS hello;",
  );
  const [result, setResult] = useState<SqlResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const execute = useCallback(async () => {
    const trimmed = query.trim();
    if (!trimmed) return;

    setLoading(true);
    setError(null);
    setResult(null);

    try {
      const res = await executeSQL(trimmed);
      setResult(res);
      localStorage.setItem("ayb_sql_query", trimmed);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError(String(err));
      }
    } finally {
      setLoading(false);
    }
  }, [query]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
        e.preventDefault();
        execute();
      }
    },
    [execute],
  );

  return (
    <div className="flex flex-col h-full">
      {/* Editor area */}
      <div className="border-b p-4">
        <textarea
          ref={textareaRef}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          className="w-full h-32 p-3 font-mono text-sm border rounded-lg resize-y bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          placeholder="Enter SQL query..."
          spellCheck={false}
        />
        <div className="mt-2 flex items-center gap-3">
          <button
            onClick={execute}
            disabled={loading || !query.trim()}
            className="px-4 py-1.5 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5"
          >
            <Play className="w-3.5 h-3.5" />
            {loading ? "Running..." : "Execute"}
          </button>
          <span className="text-xs text-gray-400">
            {navigator.platform.includes("Mac") ? "\u2318" : "Ctrl"}+Enter to
            run
          </span>
          {result && (
            <span className="ml-auto text-xs text-gray-500 flex items-center gap-1">
              <Clock className="w-3 h-3" />
              {result.rowCount} row{result.rowCount !== 1 ? "s" : ""} in{" "}
              {result.durationMs}ms
            </span>
          )}
        </div>
      </div>

      {/* Results area */}
      <div className="flex-1 overflow-auto">
        {error && (
          <div className="m-4 p-3 bg-red-50 border border-red-200 rounded-lg flex items-start gap-2">
            <AlertCircle className="w-4 h-4 text-red-500 mt-0.5 shrink-0" />
            <pre className="text-sm text-red-700 whitespace-pre-wrap font-mono">
              {error}
            </pre>
          </div>
        )}

        {result && result.columns.length > 0 && (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-gray-50">
                  {result.columns.map((col) => (
                    <th
                      key={col}
                      className="px-4 py-2 text-left font-medium text-gray-600 whitespace-nowrap"
                    >
                      {col}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {result.rows.map((row, i) => (
                  <tr key={i} className="border-b hover:bg-gray-50">
                    {row.map((cell, j) => (
                      <td
                        key={j}
                        className="px-4 py-2 whitespace-nowrap font-mono text-xs"
                      >
                        {cell === null ? (
                          <span className="text-gray-300 italic">null</span>
                        ) : typeof cell === "object" ? (
                          JSON.stringify(cell)
                        ) : (
                          String(cell)
                        )}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {result && result.columns.length === 0 && (
          <div className="m-4 text-sm text-gray-500">
            Query executed successfully. {result.rowCount} row
            {result.rowCount !== 1 ? "s" : ""} affected.
          </div>
        )}

        {!result && !error && (
          <div className="flex-1 flex items-center justify-center text-gray-400 text-sm h-48">
            Run a query to see results
          </div>
        )}
      </div>
    </div>
  );
}
