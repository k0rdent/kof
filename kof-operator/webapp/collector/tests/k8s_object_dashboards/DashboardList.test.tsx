import { render, screen, fireEvent } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { FakeDashboardData } from "../fake_data/fake_k8s_object_data";
import DashboardList from "../../src/components/pages/dashboards/DashboardList";
import "@testing-library/jest-dom";

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

describe("DashboardList", () => {
  it("should render icon and health summary", () => {
    const dashboardData = FakeDashboardData();
    render(<DashboardList {...dashboardData} />);

    expect(document.querySelector("svg.lucide-settings")).toBeInTheDocument();
    expect(screen.getByText("2 Total")).toBeInTheDocument();
    expect(screen.getByText("1 healthy")).toBeInTheDocument();
    expect(screen.getByText("1 unhealthy")).toBeInTheDocument();
  });

  it("should render the table headers and rows", () => {
    const dashboardData = FakeDashboardData();
    render(<DashboardList {...dashboardData} />);

    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("Status")).toBeInTheDocument();
    expect(screen.getByText("Age")).toBeInTheDocument();
    expect(screen.getByText("dashboard-1")).toBeInTheDocument();
    expect(screen.getByText("healthy")).toBeInTheDocument();
    expect(screen.getByText("dashboard-2")).toBeInTheDocument();
    expect(screen.getByText("unhealthy")).toBeInTheDocument();
  });

  it("should navigate on row click", async () => {
    const dashboardData = FakeDashboardData();
    render(<DashboardList {...dashboardData} />);

    const row = screen.getByText("dashboard-1").closest("tr");
    fireEvent.click(row!);
    expect(mockNavigate).toHaveBeenCalledWith("cluster-1/ns1/dashboard-1");
  });

  it("should render empty state if no items", () => {
    const dashboardData = FakeDashboardData({
      name: "FakeObjects",
      store: () => ({
        isLoading: false,
        items: null,
        selectedItem: null,
        error: null,
        selectItem: () => undefined,
        fetch: vi.fn(),
      }),
    });

    render(<DashboardList {...dashboardData} />);
    expect(screen.queryByText("dashboard-1")).not.toBeInTheDocument();
    expect(screen.queryByText("dashboard-2")).not.toBeInTheDocument();
  });
});
