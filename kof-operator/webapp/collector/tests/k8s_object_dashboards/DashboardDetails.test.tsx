import DashboardDetails from "../../src/components/pages/dashboards/DashboardDetails";
import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, cleanup } from "@testing-library/react";
import {
  MockK8sObjects,
  FakeK8sObject,
  FakeDashboardData,
  basicTabs,
} from "../fake_data/fake_k8s_object_data";
import { DetailsNewTab } from "../../src/components/pages/dashboards/DetailTabs";

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useNavigate: () => vi.fn(),
  };
});

describe("DashboardDetails", () => {
  afterEach(() => {
    cleanup();
  });

  it("should show not found state and back button", () => {
    const dashboardData = FakeDashboardData({
      store: () => ({
        isLoading: false,
        items: null,
        selectedItem: null,
        selectItem: () => undefined,
        error: null,
        fetch: vi.fn(),
      }),
    });

    render(<DashboardDetails {...dashboardData} />);
    expect(screen.getByText(/Test Dashboard not found/i)).toBeInTheDocument();
    expect(screen.getByText("Back to Table")).toBeInTheDocument();
  });

  it("should render details header and tabs", () => {
    const dashboardData = FakeDashboardData();
    const selectedItem = dashboardData.store().selectedItem!;
    render(<DashboardDetails {...dashboardData} />);
    expect(document.querySelector("svg.lucide-settings")).toBeInTheDocument();
    expect(screen.getByText(selectedItem.name)).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Status" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Metadata" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Raw Json" })).toBeInTheDocument();
  });

  it("should render custom tab", () => {
    const customTab = DetailsNewTab("Custom Tab", (item: FakeK8sObject) => (
      <div>Hello {item.name}</div>
    ));
    const dashboardData = FakeDashboardData({
      detailTabs: [customTab, ...basicTabs],
    });

    render(<DashboardDetails {...dashboardData} />);
    expect(screen.getByRole("tab", { name: "Status" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Metadata" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Raw Json" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Custom Tab" })).toBeInTheDocument();
  });

  it("should disable Metadata tab if isDisabledFn returns true", () => {
    const selectedItem = MockK8sObjects[0];
    const customDisabledTab = DetailsNewTab(
      "TestTab",
      (item: FakeK8sObject) => <div>{item.name}</div>,
      (item: FakeK8sObject) => item.name == selectedItem.name,
    );
    const dashboardData = FakeDashboardData({
      detailTabs: [customDisabledTab, ...basicTabs],
    });

    render(<DashboardDetails {...dashboardData} />);
    expect(screen.getByRole("tab", { name: "Status" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Metadata" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Raw Json" })).toBeInTheDocument();
    // The "TestTab" should not be rendered because isDisabledFn returns true for the selected item.
    expect(screen.queryByRole("tab", { name: "TestTab" })).not.toBeInTheDocument();
  });
});
