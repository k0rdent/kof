import "@testing-library/jest-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, renderHook, screen } from "@testing-library/react";
import {
  MetricsCard,
  MetricCardRow,
  CustomRowProps,
} from "../src/components/shared/MetricsCard";
import React, { act } from "react";
import { useVictoriaMetricsState } from "../src/providers/victoria_metrics/VictoriaMetricsProvider";
import { fakeVictoriaResponse } from "./fake_data/fake_victoria_response";
import { VICTORIA_METRICS } from "../src/constants/metrics.constants";

describe("Metrics Card", () => {
  const MockIcon = React.forwardRef<SVGSVGElement>((props, ref) => (
    <svg ref={ref} data-testid="mock-icon" />
  ));

  beforeEach(async () => {
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

    vi.mock("../src/providers/collectors_metrics/TimePeriodState", () => ({
      useTimePeriod: () => ({ timePeriod: "1h" }),
    }));

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

  it("should renders nothing if no rows array is empty", async () => {
    const title = "No Rows";

    render(
      <MetricsCard
        rows={[]}
        icon={MockIcon}
        title={title}
        state={useVictoriaMetricsState}
      />
    );

    expect(screen.getByText(title)).toBeInTheDocument();

    const cardContent = document.querySelector('div[data-slot="card-content"]');
    expect(cardContent).toBeEmptyDOMElement();
  });

  it("should render card row with metricFetchFn", () => {
    const cardTitle = "Test Card";

    const rows: MetricCardRow[] = [
      {
        title: "Custom Metric",
        metricFetchFn: () => 123,
        metricFormat: (v) => `${v} units`,
      },
    ];
    render(
      <MetricsCard
        rows={rows}
        icon={MockIcon}
        title={cardTitle}
        state={useVictoriaMetricsState}
      />
    );

    expect(screen.getByText(cardTitle)).toBeInTheDocument();

    const cardContent = document.querySelector('div[data-slot="card-content"]');
    expect(cardContent).not.toBeEmptyDOMElement();

    const contentRows = cardContent?.childNodes;
    expect(contentRows).toHaveLength(1);

    expect(screen.getByText("Custom Metric")).toBeInTheDocument();
    expect(screen.getByText("123 units")).toBeInTheDocument();
  });

  it("should render card row with metricName and no trend", () => {
    const cardTitle = "Rows Card";

    const rows: MetricCardRow[] = [
      {
        title: "HTTP Requests",
        metricName: VICTORIA_METRICS.VM_HTTP_REQUESTS_TOTAL,
      },
    ];

    render(
      <MetricsCard
        rows={rows}
        icon={MockIcon}
        title={cardTitle}
        state={useVictoriaMetricsState}
      />
    );

    expect(screen.getByText(cardTitle)).toBeInTheDocument();

    const cardContent = document.querySelector('div[data-slot="card-content"]');
    expect(cardContent).not.toBeEmptyDOMElement();

    const contentRows = cardContent?.childNodes;
    expect(contentRows).toHaveLength(1);

    expect(screen.getByText("HTTP Requests")).toBeInTheDocument();
    expect(screen.getByText("39107")).toBeInTheDocument();
  });

  it("should render card row with enableTrendSystem", () => {
    const cardTitle = "CPU";

    const rows: MetricCardRow[] = [
      {
        title: "Seconds Total",
        metricName: VICTORIA_METRICS.GO_GC_CPU_SECONDS_TOTAL,
        enableTrendSystem: true,
        metricFormat: (v) => `${v}s`,
      },
    ];

    render(
      <MetricsCard
        rows={rows}
        icon={MockIcon}
        title={cardTitle}
        state={useVictoriaMetricsState}
      />
    );

    expect(screen.getByText(cardTitle)).toBeInTheDocument();
    expect(screen.getByText("16.343930724s")).toBeInTheDocument();
  });

  it("should render card with custom row", () => {
    const cardTitle = "Custom Row Card";

    const CustomRow = ({ formattedValue, title }: CustomRowProps) => (
      <div data-testid="custom-row">
        {title}: {formattedValue}
      </div>
    );

    const rows: MetricCardRow[] = [
      {
        title: "Custom Row",
        metricFetchFn: () => 77,
        metricFormat: (v) => `${v}x`,
        customRow: CustomRow,
      },
    ];

    render(
      <MetricsCard
        rows={rows}
        icon={MockIcon}
        title={cardTitle}
        state={useVictoriaMetricsState}
      />
    );

    expect(screen.getByText(cardTitle)).toBeInTheDocument();
    expect(screen.getByTestId("custom-row")).toHaveTextContent(
      "Custom Row: 77x"
    );
  });

  it("should render card with description", () => {
    const cardTitle = "Desc Card";

    render(
      <MetricsCard
        rows={[]}
        icon={MockIcon}
        title={cardTitle}
        state={useVictoriaMetricsState}
        description="This is a description"
      />
    );

    expect(screen.getByText(cardTitle)).toBeInTheDocument();
    expect(screen.getByText("This is a description")).toBeInTheDocument();
  });

  it("should render card with multiple rows correctly", () => {
    const cardTitle = "Multiple Rows Card";
    const rows: MetricCardRow[] = [
      {
        title: "Row 1",
        metricFetchFn: () => 10,
        metricFormat: (v) => `${v}%`,
      },
      {
        title: "Row 2",
        metricName: VICTORIA_METRICS.VM_HTTP_REQUESTS_TOTAL,
      },
      {
        title: "Row 3",
        metricFetchFn: () => 42,
        metricFormat: (v) => `${v}s`,
      },
    ];

    render(
      <MetricsCard
        rows={rows}
        icon={MockIcon}
        title={cardTitle}
        state={useVictoriaMetricsState}
      />
    );

    expect(screen.getByText(cardTitle)).toBeInTheDocument();

    const cardContent = document.querySelector('div[data-slot="card-content"]');
    expect(cardContent?.childNodes).toHaveLength(3);

    expect(screen.getByText("Row 1")).toBeInTheDocument();
    expect(screen.getByText("10%")).toBeInTheDocument();

    expect(screen.getByText("Row 2")).toBeInTheDocument();
    expect(screen.getByText("39107")).toBeInTheDocument();

    expect(screen.getByText("Row 3")).toBeInTheDocument();
    expect(screen.getByText("42s")).toBeInTheDocument();
  });
});
