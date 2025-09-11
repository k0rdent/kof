import DashboardLayout from "../../src/components/pages/dashboards/DashboardLayout";
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { FakeDashboardData } from "../fake_data/fake_k8s_object_data";

describe("DashboardLayout", () => {
  it("should render loader when isLoading is true", () => {
    const dashboardData = FakeDashboardData({
      name: "FakeObjects",
      store: () => ({
        isLoading: true,
        items: null,
        selectedItem: null,
        error: null,
        selectItem: () => undefined,
        fetch: vi.fn(),
      }),
    });

    render(<DashboardLayout {...dashboardData} />);
    expect(document.querySelector("svg.lucide-loader")).toBeInTheDocument();
  });

  it("should render error message when error is present", () => {
    const dashboardData = FakeDashboardData({
      name: "FakeObjects",
      store: () => ({
        isLoading: false,
        items: null,
        selectedItem: null,
        error: new Error("Network error"),
        selectItem: () => undefined,
        fetch: vi.fn(),
      }),
    });

    render(<DashboardLayout {...dashboardData} />);
    expect(screen.getByText(/Failed to fetch FakeObjects/i)).toBeInTheDocument();
    expect(screen.getByText("Reload")).toBeInTheDocument();
  });

  it("should render no items message when items is empty", () => {
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

    render(<DashboardLayout {...dashboardData} />);
    expect(screen.getByText(/No FakeObjects found/i)).toBeInTheDocument();
  });

  it("should render Outlet when items are present", () => {
    const dashboardData = FakeDashboardData({
      name: "FakeObjects",
    });

    render(<DashboardLayout {...dashboardData} />);
    expect(screen.getByText("FakeObjects")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "FakeObjects" })).toBeInTheDocument();
  });
});
