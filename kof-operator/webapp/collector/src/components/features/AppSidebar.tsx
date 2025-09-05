import { ForwardRefExoticComponent, JSX, RefAttributes } from "react";
import {
  Target,
  Funnel,
  Database,
  Server,
  TriangleAlert,
  LucideProps,
  Layers,
  Workflow,
  ServerCog,
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
import { useClusterDeploymentsProvider } from "@/providers/ClusterDeploymentsProvider";
import { useClusterSummariesProvider } from "@/providers/ClusterSummariesProvider";
import { useMultiClusterServiceProvider } from "@/providers/MultiClusterServicesProvider";
import { useStateManagementProvidersProvider } from "@/providers/StateManagementProvidersProvider";

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
    items: clusterDeployment,
    isLoading: isClusterLoading,
    error: clusterError,
  } = useClusterDeploymentsProvider();

  const {
    items: summaries,
    isLoading: isSummariesLoading,
    error: summariesError,
  } = useClusterSummariesProvider();

  const {
    items: services,
    isLoading: isServicesLoading,
    error: servicesError
  } = useMultiClusterServiceProvider();

  const {
    items: SMPs,
    isLoading: isSMPsLoading,
    error: SMPsError
  } = useStateManagementProvidersProvider()

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
      alert: clusterDeployment
        ? !isClusterLoading && !clusterError && !clusterDeployment.isHealthy
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
    {
      title: "Multi Cluster Services",
      url: "multi-cluster-services",
      icon: Workflow,
      alert: services
        ? !isServicesLoading && !servicesError && !services.isHealthy
        : false,
    },
    {
      title: "State Management Providers",
      url: "state-management-providers",
      icon: ServerCog,
      alert: SMPs
        ? !isSMPsLoading && !SMPsError && !SMPs.isHealthy
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
