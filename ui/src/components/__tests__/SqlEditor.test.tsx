import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MockApiError } from "../../test-utils";

vi.mock("../../api", () => ({
  executeSQL: vi.fn(),
  ApiError: MockApiError,
}));

import { SqlEditor } from "../SqlEditor";
import { executeSQL } from "../../api";
const mockExecuteSQL = vi.mocked(executeSQL);

describe("SqlEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  it("renders with default query", () => {
    render(<SqlEditor />);

    const textarea = screen.getByPlaceholderText("Enter SQL query...");
    expect(textarea).toBeInTheDocument();
    expect(textarea).toHaveValue("SELECT 1 AS hello;");
    expect(screen.getByRole("button", { name: /Execute/ })).toBeInTheDocument();
  });

  it("restores query from localStorage", () => {
    localStorage.setItem("ayb_sql_query", "SELECT * FROM users;");
    render(<SqlEditor />);

    expect(screen.getByPlaceholderText("Enter SQL query...")).toHaveValue(
      "SELECT * FROM users;",
    );
  });

  it("executes query and displays results", async () => {
    mockExecuteSQL.mockResolvedValueOnce({
      columns: ["id", "name"],
      rows: [
        [1, "alice"],
        [2, "bob"],
      ],
      rowCount: 2,
      durationMs: 5,
    });
    const user = userEvent.setup();

    render(<SqlEditor />);
    await user.click(screen.getByRole("button", { name: /Execute/ }));

    await waitFor(() => {
      expect(screen.getByText("id")).toBeInTheDocument();
      expect(screen.getByText("name")).toBeInTheDocument();
      expect(screen.getByText("alice")).toBeInTheDocument();
      expect(screen.getByText("bob")).toBeInTheDocument();
      expect(screen.getByText(/2 rows in 5ms/)).toBeInTheDocument();
    });
  });

  it("shows error on query failure", async () => {
    const { ApiError } = await import("../../api");
    mockExecuteSQL.mockRejectedValueOnce(
      new ApiError(400, 'relation "nonexistent" does not exist'),
    );
    const user = userEvent.setup();

    render(<SqlEditor />);
    await user.click(screen.getByRole("button", { name: /Execute/ }));

    await waitFor(() => {
      expect(
        screen.getByText('relation "nonexistent" does not exist'),
      ).toBeInTheDocument();
    });
  });

  it("renders null cells as italic null", async () => {
    mockExecuteSQL.mockResolvedValueOnce({
      columns: ["name"],
      rows: [[null]],
      rowCount: 1,
      durationMs: 1,
    });
    const user = userEvent.setup();

    render(<SqlEditor />);
    await user.click(screen.getByRole("button", { name: /Execute/ }));

    await waitFor(() => {
      expect(screen.getByText("null")).toBeInTheDocument();
    });
  });

  it("shows placeholder when no results", () => {
    render(<SqlEditor />);
    expect(screen.getByText("Run a query to see results")).toBeInTheDocument();
  });

  it("shows affected rows for DDL/DML without columns", async () => {
    mockExecuteSQL.mockResolvedValueOnce({
      columns: [],
      rows: [],
      rowCount: 3,
      durationMs: 2,
    });
    const user = userEvent.setup();

    render(<SqlEditor />);
    await user.click(screen.getByRole("button", { name: /Execute/ }));

    await waitFor(() => {
      expect(
        screen.getByText(/Query executed successfully. 3 rows affected./),
      ).toBeInTheDocument();
    });
  });

  it("saves query to localStorage on successful execution", async () => {
    mockExecuteSQL.mockResolvedValueOnce({
      columns: ["x"],
      rows: [[1]],
      rowCount: 1,
      durationMs: 1,
    });
    const user = userEvent.setup();

    render(<SqlEditor />);
    const textarea = screen.getByPlaceholderText("Enter SQL query...");
    await user.clear(textarea);
    await user.type(textarea, "SELECT 42;");
    await user.click(screen.getByRole("button", { name: /Execute/ }));

    await waitFor(() => {
      expect(localStorage.getItem("ayb_sql_query")).toBe("SELECT 42;");
    });
  });

  it("does not execute empty query", async () => {
    const user = userEvent.setup();

    render(<SqlEditor />);
    const textarea = screen.getByPlaceholderText("Enter SQL query...");
    await user.clear(textarea);

    // Button should be disabled when query is empty.
    expect(screen.getByRole("button", { name: /Execute/ })).toBeDisabled();
    expect(mockExecuteSQL).not.toHaveBeenCalled();
  });

  it("renders JSON object cells as stringified JSON", async () => {
    mockExecuteSQL.mockResolvedValueOnce({
      columns: ["data"],
      rows: [[{ key: "value" }]],
      rowCount: 1,
      durationMs: 1,
    });
    const user = userEvent.setup();

    render(<SqlEditor />);
    await user.click(screen.getByRole("button", { name: /Execute/ }));

    await waitFor(() => {
      expect(screen.getByText('{"key":"value"}')).toBeInTheDocument();
    });
  });

  it("executes query with Ctrl+Enter keyboard shortcut", async () => {
    mockExecuteSQL.mockResolvedValueOnce({
      columns: ["x"],
      rows: [[1]],
      rowCount: 1,
      durationMs: 1,
    });
    const user = userEvent.setup();

    render(<SqlEditor />);
    const textarea = screen.getByPlaceholderText("Enter SQL query...");
    await user.click(textarea);
    await user.keyboard("{Control>}{Enter}{/Control}");

    await waitFor(() => {
      expect(mockExecuteSQL).toHaveBeenCalledWith("SELECT 1 AS hello;");
    });
  });

  it("singular row text for 1 row", async () => {
    mockExecuteSQL.mockResolvedValueOnce({
      columns: ["x"],
      rows: [[1]],
      rowCount: 1,
      durationMs: 10,
    });
    const user = userEvent.setup();

    render(<SqlEditor />);
    await user.click(screen.getByRole("button", { name: /Execute/ }));

    await waitFor(() => {
      expect(screen.getByText(/1 row in 10ms/)).toBeInTheDocument();
    });
  });
});
