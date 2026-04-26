import {
	Sidebar,
	SidebarContent,
	SidebarFooter,
	SidebarGroup,
	SidebarGroupContent,
	SidebarGroupLabel,
	SidebarHeader,
	SidebarMenu,
	SidebarMenuButton,
	SidebarMenuItem,
	SidebarRail,
} from "@/components/ui/sidebar";
import { useAuth } from "@/contexts/AuthContext";
import {
	Activity,
	BookTemplate,
	LayoutDashboard,
	LogOut,
	Server,
	FolderGit2,
	Briefcase,
	Key,
	Calendar,
	GitBranch,
} from "lucide-react";
import { Link, useLocation } from "react-router-dom";

const navItems = [
	{ title: "Dashboard", url: "/dashboard", icon: LayoutDashboard },
	{ title: "Jobs", url: "/jobs", icon: Activity },
	{ title: "Job Templates", url: "/templates", icon: BookTemplate },
	{ title: "Projects", url: "/projects", icon: FolderGit2 },
	{ title: "Agents", url: "/agents", icon: Server },
];

const deferredItems = [
	{ title: "Credentials", url: "/credentials", icon: Key },
	{ title: "Schedules", url: "/schedules", icon: Calendar },
	{ title: "Workflows", url: "/workflows", icon: GitBranch },
];

export function AppSidebar() {
	const { logout } = useAuth();
	const { pathname } = useLocation();

	return (
		<Sidebar variant="inset">
			<SidebarHeader className="px-4 py-3">
				<div className="flex items-center gap-2">
					<Briefcase className="size-5 text-primary" />
					<span className="font-semibold text-sm tracking-wide">AOP</span>
				</div>
			</SidebarHeader>

			<SidebarContent>
				<SidebarGroup>
					<SidebarGroupContent>
						<SidebarMenu>
							{navItems.map((item) => (
								<SidebarMenuItem key={item.url}>
									<SidebarMenuButton
										asChild
										isActive={
											item.url === "/dashboard"
												? pathname === "/" || pathname === "/dashboard"
												: pathname.startsWith(item.url)
										}
									>
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

				<SidebarGroup>
					<SidebarGroupLabel>Coming Soon</SidebarGroupLabel>
					<SidebarGroupContent>
						<SidebarMenu>
							{deferredItems.map((item) => (
								<SidebarMenuItem key={item.url}>
									<SidebarMenuButton
										asChild
										className="text-muted-foreground"
									>
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

			<SidebarFooter>
				<SidebarMenu>
					<SidebarMenuItem>
						<SidebarMenuButton onClick={logout} className="text-muted-foreground">
							<LogOut />
							<span>Sign out</span>
						</SidebarMenuButton>
					</SidebarMenuItem>
				</SidebarMenu>
			</SidebarFooter>

			<SidebarRail />
		</Sidebar>
	);
}
