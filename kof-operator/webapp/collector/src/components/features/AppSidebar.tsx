import { JSX } from "react";
import { Target, Funnel } from "lucide-react";
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

const items = [
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
];

const AppSidebar = (): JSX.Element => {
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
                    <Link to={item.url}>
                      <item.icon />
                      <span>{item.title}</span>
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
