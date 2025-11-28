import "@testing-library/jest-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { fakeVictoriaResponse } from "../fake_data/fake_victoria_response";
import { render, renderHook, screen, act } from "@testing-library/react";
import VictoriaList from "../../src/components/pages/victoriaPage/victoria-list/VictoriaList";
import { useVictoriaMetricsState } from "../../src/providers/victoria_metrics/VictoriaMetricsProvider";

describe("Victoria List", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(fakeVictoriaResponse),
      }),
    );

    vi.mock("../../src/metrics/MetricsDatabase", () => {
      return {
        MetricsDatabase: vi.fn(),
      };
    });

    vi.mock("react-router-dom", async () => {
      const actual = await vi.importActual("react-router-dom");
      return {
        ...actual,
        useNavigate: () => vi.fn(),
      };
    });
  });

  afterEach(async () => {
    vi.resetAllMocks();
    act(() => useVictoriaMetricsState.setState({
      isLoading: false,
      data: null,
      selectedCluster: null,
      selectedPod: null,
      error: undefined,
    }));
    await act(async () => await Promise.resolve());
  });

  it("should render victoria list with data", async () => {
    const { result } = renderHook(() => useVictoriaMetricsState());
    await result.current.fetch();

    await act(() => render(<VictoriaList />));

    expect(screen.getByText("mothership")).toBeInTheDocument();
    expect(screen.getByText("aws-ue2")).toBeInTheDocument();
    expect(result.current.data).not.toBeNull();
    expect(result.current.error).toBeUndefined();
    expect(result.current.isLoading).toBeFalsy();
  });

  it("should render victoria list without data", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ clusters: {} }),
      }),
    );
    const { result } = renderHook(() => useVictoriaMetricsState());

    await act(() => render(<VictoriaList />));

    expect(screen.getByText("No clusters found")).toBeInTheDocument();
    expect(result.current.data).toBeNull();
    expect(result.current.error).toBeUndefined();
    expect(result.current.isLoading).toBeFalsy();
  });

  it("should show loading spinner when data is loading", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => {
          new Promise(function (resolve) {
            setTimeout(resolve, 5000);
          });
        },
      }),
    );

    const { result } = renderHook(() => useVictoriaMetricsState());
    result.current.fetch();

    render(<VictoriaList />);

    const spinner = document.querySelector(".animate-spin");
    expect(spinner).toBeInTheDocument();
    expect(result.current.isLoading).toBeTruthy();
    expect(result.current.data).toBeNull();
    expect(result.current.error).toBeUndefined();
  });

  it("should handle fetch errors", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("Network error")));

    const { result } = renderHook(() => useVictoriaMetricsState());
    await result.current.fetch();

    await act(() => render(<VictoriaList />));

    expect(
      screen.getByText(
        `Failed to fetch collectors metrics. Click "Reload" button to try again.`,
      ),
    ).toBeInTheDocument();
    expect(result.current.isLoading).toBe(false);
    expect(result.current.data).toBeNull();
    expect(result.current.error).toBeInstanceOf(Error);
  });

  it("should render two tables for two clusters", async () => {
    const { result } = renderHook(() => useVictoriaMetricsState());
    await result.current.fetch();

    await act(() => render(<VictoriaList />));

    const tables = document.querySelectorAll("table");
    expect(tables).toHaveLength(2);

    const awsUe2ClusterRows = tables[0].querySelectorAll("tbody tr");
    expect(awsUe2ClusterRows.length).toBe(2);

    const mothershipClusterRows = tables[1].querySelectorAll("tbody tr");
    expect(mothershipClusterRows.length).toBe(3);
  });
});
