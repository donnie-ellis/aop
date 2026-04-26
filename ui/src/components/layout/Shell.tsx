import { AppSidebar } from "@/components/layout/AppSidebar";
import { SidebarInset, SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar";
import { useAuth } from "@/contexts/AuthContext";
import { Navigate, Outlet } from "react-router-dom";

export function Shell() {
	const { token } = useAuth();
	if (!token) return <Navigate to="/login" replace />;

	return (
		<SidebarProvider>
			<AppSidebar />
			<SidebarInset>
				<header className="flex h-12 shrink-0 items-center gap-2 border-b px-4">
					<SidebarTrigger className="-ml-1" />
				</header>
				<div className="flex flex-1 flex-col gap-4 p-6 overflow-auto">
					<Outlet />
				</div>
			</SidebarInset>
		</SidebarProvider>
	);
}
