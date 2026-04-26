import { Shell } from "@/components/layout/Shell";
import { AgentsPage } from "@/pages/agents/AgentsPage";
import { DashboardPage } from "@/pages/dashboard/DashboardPage";
import { JobDetailPage } from "@/pages/jobs/JobDetailPage";
import { JobsPage } from "@/pages/jobs/JobsPage";
import { LoginPage } from "@/pages/login/LoginPage";
import { ProjectsPage } from "@/pages/projects/ProjectsPage";
import { ComingSoonPage } from "@/pages/stubs/ComingSoonPage";
import { TemplatesPage } from "@/pages/templates/TemplatesPage";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";

export function App() {
	return (
		<BrowserRouter>
			<Routes>
				<Route path="/login" element={<LoginPage />} />
				<Route element={<Shell />}>
					<Route path="/" element={<Navigate to="/dashboard" replace />} />
					<Route path="/dashboard" element={<DashboardPage />} />
					<Route path="/jobs" element={<JobsPage />} />
					<Route path="/jobs/:id" element={<JobDetailPage />} />
					<Route path="/templates" element={<TemplatesPage />} />
					<Route path="/projects" element={<ProjectsPage />} />
					<Route path="/agents" element={<AgentsPage />} />
					<Route
						path="/credentials"
						element={<ComingSoonPage title="Credentials" />}
					/>
					<Route
						path="/schedules"
						element={<ComingSoonPage title="Schedules" />}
					/>
					<Route
						path="/workflows"
						element={<ComingSoonPage title="Workflows" />}
					/>
				</Route>
			</Routes>
		</BrowserRouter>
	);
}
