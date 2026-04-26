import { StatusBadge } from "@/components/StatusBadge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { useJobs } from "@/hooks/useJobs";
import type { JobStatus } from "@/types";
import { formatDistanceToNow } from "date-fns";
import { useState } from "react";
import { Link } from "react-router-dom";

const STATUS_OPTIONS: { value: JobStatus | "all"; label: string }[] = [
	{ value: "all", label: "All statuses" },
	{ value: "pending", label: "Pending" },
	{ value: "dispatched", label: "Dispatched" },
	{ value: "running", label: "Running" },
	{ value: "success", label: "Success" },
	{ value: "failed", label: "Failed" },
	{ value: "cancelled", label: "Cancelled" },
];

export function JobsPage() {
	const [statusFilter, setStatusFilter] = useState<JobStatus | "all">("all");
	const { data: jobs, isLoading } = useJobs(
		statusFilter !== "all" ? { status: statusFilter } : {},
	);

	return (
		<div className="space-y-4">
			<div className="flex items-center justify-between">
				<h1 className="text-2xl font-semibold">Jobs</h1>
				<Select
					value={statusFilter}
					onValueChange={(v) => setStatusFilter(v as JobStatus | "all")}
				>
					<SelectTrigger className="w-44">
						<SelectValue />
					</SelectTrigger>
					<SelectContent>
						{STATUS_OPTIONS.map((o) => (
							<SelectItem key={o.value} value={o.value}>
								{o.label}
							</SelectItem>
						))}
					</SelectContent>
				</Select>
			</div>

			<Card>
				<CardContent className="p-0">
					{isLoading ? (
						<div className="space-y-2 p-6">
							{[...Array(8)].map((_, i) => (
								// biome-ignore lint/suspicious/noArrayIndexKey: skeleton
								<Skeleton key={i} className="h-10 w-full" />
							))}
						</div>
					) : !jobs?.length ? (
						<p className="py-12 text-center text-sm text-muted-foreground">
							No jobs found.
						</p>
					) : (
						<table className="w-full text-sm">
							<thead>
								<tr className="border-b text-left text-muted-foreground">
									<th className="px-6 py-3 font-medium">ID</th>
									<th className="px-6 py-3 font-medium">Status</th>
									<th className="px-6 py-3 font-medium">Agent</th>
									<th className="px-6 py-3 font-medium">Created</th>
									<th className="px-6 py-3 font-medium">Duration</th>
									<th className="px-6 py-3" />
								</tr>
							</thead>
							<tbody>
								{jobs.map((job) => {
									const duration =
										job.started_at && job.ended_at
											? Math.round(
													(new Date(job.ended_at).getTime() -
														new Date(job.started_at).getTime()) /
														1000,
												)
											: null;

									return (
										<tr
											key={job.id}
											className="border-b last:border-0 hover:bg-muted/50"
										>
											<td className="px-6 py-3">
												<Link
													to={`/jobs/${job.id}`}
													className="font-mono text-xs hover:underline"
												>
													{job.id.slice(0, 8)}…
												</Link>
											</td>
											<td className="px-6 py-3">
												<StatusBadge status={job.status} />
											</td>
											<td className="px-6 py-3 font-mono text-xs text-muted-foreground">
												{job.agent_id ? job.agent_id.slice(0, 8) : "—"}
											</td>
											<td className="px-6 py-3 text-xs text-muted-foreground">
												{formatDistanceToNow(new Date(job.created_at), {
													addSuffix: true,
												})}
											</td>
											<td className="px-6 py-3 text-xs text-muted-foreground">
												{duration != null ? `${duration}s` : "—"}
											</td>
											<td className="px-6 py-3 text-right">
												<Button variant="ghost" size="sm" asChild>
													<Link to={`/jobs/${job.id}`}>Logs</Link>
												</Button>
											</td>
										</tr>
									);
								})}
							</tbody>
						</table>
					)}
				</CardContent>
			</Card>
		</div>
	);
}
