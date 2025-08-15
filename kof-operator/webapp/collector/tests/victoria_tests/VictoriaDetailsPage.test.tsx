import "@testing-library/jest-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { fakeVictoriaResponse } from "../fake_data/fake_victoria_response";
import { act, render, renderHook, screen } from "@testing-library/react";
import VictoriaDetailsPage from "../../src/components/pages/victoriaPage/victoria-details/VictoriaDetailsPage";
import { useVictoriaMetricsState } from "../../src/providers/victoria_metrics/VictoriaMetricsProvider";

describe("Victoria List", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(fakeVictoriaResponse),
      })
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

    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.resetAllMocks();
    vi.useRealTimers();
    useVictoriaMetricsState.setState({
      isLoading: false,
      data: null,
      selectedCluster: null,
      selectedPod: null,
      error: undefined,
    });
  });

  it("should render victoria details page", async () => {
    const { result } = renderHook(() => useVictoriaMetricsState());
    await result.current.fetch();

    await act(async () => {
      result.current.setSelectedCluster("aws-ue2");
    });

    await act(async () => {
      result.current.setSelectedPod(
        result.current.selectedCluster?.pods[0].name || ""
      );
    });

    const selectedPod = result.current.selectedPod;

    render(<VictoriaDetailsPage />);

    expect(
      screen.getByText(`VictoriaLogs Select: ${selectedPod.name}`)
    ).toBeInTheDocument();

    const tablist = screen.getByRole("tablist");
    expect(tablist).toBeInTheDocument();
    expect(tablist.querySelectorAll("button")).toHaveLength(5);

    expect(screen.getByText("Overview")).toBeInTheDocument();
    expect(screen.getByText("System")).toBeInTheDocument();
    expect(screen.getByText("Go Runtime")).toBeInTheDocument();
    expect(screen.getByText("Network")).toBeInTheDocument();
    expect(screen.getByText("VictoriaLogs Select")).toBeInTheDocument();
  });

  it("should show loading spinner when data is loading", () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => {
          new Promise((resolve) => setTimeout(resolve, 5000));
        },
      })
    );

    const { result } = renderHook(() => useVictoriaMetricsState());
    result.current.fetch();

    render(<VictoriaDetailsPage />);

    const spinner = document.querySelector(".animate-spin");
    expect(spinner).toBeInTheDocument();
    expect(result.current.isLoading).toBeTruthy();
    expect(result.current.data).toBeNull();
    expect(result.current.error).toBeUndefined();
    vi.runAllTimers();
  });

  it("should handle correctly if pod not found", async () => {
    const { result } = renderHook(() => useVictoriaMetricsState());
    await result.current.fetch();

    render(<VictoriaDetailsPage />);

    expect(screen.getByText("Victoria pods not found")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Back to Table" })
    ).toBeInTheDocument();
  });
});
