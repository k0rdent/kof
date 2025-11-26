import { ForwardRefExoticComponent, JSX, RefAttributes } from "react";
import { Target, Funnel, Database, TriangleAlert, LucideProps } from "lucide-react";
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/generated/ui/sidebar";
import { Link } from "react-router-dom";
import { Dashboards } from "../pages/dashboards/DashboardFactories";
import { useCollectorMetricsState } from "@/providers/collectors_metrics/CollectorsMetricsProvider";

interface SidebarItem {
  title: string;
  url: string;
  icon: ForwardRefExoticComponent<
    Omit<LucideProps, "ref"> & RefAttributes<SVGSVGElement>
  >;
  alert?: boolean;
}

const AppSidebar = (): JSX.Element => {
  const { data } = useCollectorMetricsState();

  const items: SidebarItem[] = [
    {
      title: "Prometheus Targets",
      url: "/",
      icon: Target,
    },
    {
      title: "Collectors Metrics",
      url: "collectors",
      icon: Funnel,
      alert: data?.clusters.some((cluster) => cluster.unhealthyPodCount > 0),
    },
    {
      title: "VictoriaMetrics/Logs",
      url: "victoria",
      icon: Database,
    },
  ];

  items.push(
    ...Dashboards.map((d) => {
      const { items, isLoading, error } = d.store();

      return {
        title: d.name,
        url: d.id,
        icon: d.icon,
        alert: items ? !isLoading && !error && !items.isHealthy : false,
      };
    }),
  );

  return (
    <Sidebar>
      <SidebarHeader>
        <h1 className="font-bold">KOF Dashboard</h1>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              {items.map((item) => (
                <SidebarMenuItem key={item.title}>
                  <SidebarMenuButton asChild>
                    <Link
                      to={item.url}
                      className="flex w-full items-center justify-between"
                    >
                      <div className="flex items-center gap-2">
                        <item.icon className="h-4 w-4" />
                        <span>{item.title}</span>
                      </div>
                      {item.alert && (
                        <TriangleAlert className="text-orange-600 w-4 h-4" />
                      )}
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>
  );
};

export default AppSidebar;
