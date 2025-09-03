import { ForwardRefExoticComponent, JSX, RefAttributes } from "react";
import {
  Target,
  Funnel,
  Database,
  Server,
  TriangleAlert,
  LucideProps,
  Layers,
} from "lucide-react";
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
import { useClusterDeploymentsProvider } from "@/providers/cluster_deployments/ClusterDeploymentsProvider";
import { useClusterSummariesProvider } from "@/providers/cluster_summaries/ClusterDeploymentsProvider";

interface SidebarItem {
  title: string;
  url: string;
  icon: ForwardRefExoticComponent<
    Omit<LucideProps, "ref"> & RefAttributes<SVGSVGElement>
  >;
  alert?: boolean;
}

const AppSidebar = (): JSX.Element => {
  const {
    data: clusterData,
    isLoading: isClusterLoading,
    error: clusterError,
  } = useClusterDeploymentsProvider();

  const {
    data: summaries,
    isLoading: isSummariesLoading,
    error: summariesError,
  } = useClusterSummariesProvider();

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
    },
    {
      title: "VictoriaMetrics/Logs",
      url: "victoria",
      icon: Database,
    },
    {
      title: "Cluster Deployments",
      url: "cluster-deployments",
      icon: Server,
      alert: clusterData
        ? !isClusterLoading && !clusterError && !clusterData.isHealthy
        : false,
    },
    {
      title: "Cluster Summaries",
      url: "cluster-summaries",
      icon: Layers,
      alert: summaries
        ? !isSummariesLoading && !summariesError && !summaries.isHealthy
        : false,
    },
  ];

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
