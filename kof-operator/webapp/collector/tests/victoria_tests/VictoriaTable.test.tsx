import "@testing-library/jest-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { fakeVictoriaResponse } from "../fake_data/fake_victoria_response";
import { render, renderHook, screen } from "@testing-library/react";
import { useVictoriaMetricsState } from "../../src/providers/victoria_metrics/VictoriaMetricsProvider";
import VictoriaTable from "../../src/components/pages/victoriaPage/victoria-list/VictoriaTable";
import { act } from "react";

describe("Victoria Table", () => {
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
  });

  afterEach(() => {
    vi.resetAllMocks();
    useVictoriaMetricsState.setState({
      isLoading: false,
      data: null,
      selectedCluster: null,
      selectedPod: null,
      error: undefined,
    });
  });

  it("should render table with cluster data", async () => {
    const { result } = renderHook(() => useVictoriaMetricsState());
    await result.current.fetch();

    await act(async () => {
      result.current.setSelectedCluster("aws-ue2");
    });

    render(<VictoriaTable cluster={result.current.selectedCluster} />);

    expect(screen.getByText("aws-ue2")).toBeInTheDocument();
    const table = document.querySelector("table");
    expect(table).toBeInTheDocument();
    expect(table?.querySelectorAll("tbody tr")).toHaveLength(1);
    expect(
      screen.getByText(
        "kof-storage-victoria-logs-cluster-vlselect-5b4fbf5c87-hbt6p"
      )
    ).toBeInTheDocument();

    const healthBadge = table?.querySelector('span[data-slot="badge"]');
    expect(healthBadge).toBeInTheDocument();
    expect(healthBadge?.textContent).toContain("healthy");
  });

  it("should render correct pod counts and health status", async () => {
    const { result } = renderHook(() => useVictoriaMetricsState());
    await result.current.fetch();

    await act(async () => {
      result.current.setSelectedCluster("mothership");
    });

    render(<VictoriaTable cluster={result.current.selectedCluster} />);

    expect(screen.getByText(/pods/)).toHaveTextContent(
      `${result.current.selectedCluster.pods.length} pods`
    );
    expect(screen.queryAllByText(/healthy/)[0]).toHaveTextContent(
      `${result.current.selectedCluster.healthyPodCount} healthy`
    );
    expect(screen.queryAllByText(/unhealthy/)[0]).toHaveTextContent(
      `${result.current.selectedCluster.unhealthyPodCount} unhealthy`
    );
  });

  it("should render table headers correctly", async () => {
    const { result } = renderHook(() => useVictoriaMetricsState());
    await result.current.fetch();

    await act(async () => {
      result.current.setSelectedCluster("aws-ue2");
    });

    render(<VictoriaTable cluster={result.current.selectedCluster} />);

    expect(screen.getByText("Pod Name")).toBeInTheDocument();
    expect(screen.getByText("Status")).toBeInTheDocument();
    expect(screen.getByText("CPU %")).toBeInTheDocument();
    expect(screen.getByText("Memory %")).toBeInTheDocument();
    expect(screen.getByText("HTTP Requests")).toBeInTheDocument();
  });

  it("should render no pods if cluster has no pods", async () => {
    const { result } = renderHook(() => useVictoriaMetricsState());

    const emptyCluster = fakeVictoriaResponse;
    emptyCluster.clusters["aws-ue2"] = {};

    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(emptyCluster),
      })
    );

    await result.current.fetch();

    await act(async () => {
      result.current.setSelectedCluster("aws-ue2");
    });

    render(<VictoriaTable cluster={result.current.selectedCluster} />);
    const table = document.querySelector("table");
    expect(table?.querySelectorAll("tbody tr")).toHaveLength(0);
    expect(screen.getByText("aws-ue2")).toBeInTheDocument();
  });
});
