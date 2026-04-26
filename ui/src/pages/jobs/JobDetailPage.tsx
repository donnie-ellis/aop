import { LogViewer } from "@/components/LogViewer";
import { StatusBadge } from "@/components/StatusBadge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useCancelJob, useJob, useJobLogs } from "@/hooks/useJobs";
import { isTerminal } from "@/types";
import { formatDistanceToNow } from "date-fns";
import { ArrowLeft, XCircle } from "lucide-react";
import { Link, useParams } from "react-router-dom";
import { toast } from "sonner";

export function JobDetailPage() {
	const { id } = useParams<{ id: string }>();
	const { data: job, isLoading: jobLoading } = useJob(id!);
	const { data: logs, isLoading: logsLoading } = useJobLogs(id!, !!id);
	const cancelJob = useCancelJob();

	const handleCancel = async () => {
		try {
			await cancelJob.mutateAsync(id!);
			toast.success("Job cancelled");
		} catch {
			toast.error("Failed to cancel job");
		}
	};

	if (jobLoading) {
		return (
			<div className="space-y-4">
				<Skeleton className="h-8 w-64" />
				<Skeleton className="h-6 w-32" />
				<Skeleton className="h-96 w-full" />
			</div>
		);
	}

	if (!job) {
		return (
			<div className="py-12 text-center text-sm text-muted-foreground">
				Job not found.
			</div>
		);
	}

	const duration =
		job.started_at && job.ended_at
			? Math.round(
					(new Date(job.ended_at).getTime() -
						new Date(job.started_at).getTime()) /
						1000,
				)
			: null;

	return (
		<div className="flex flex-col h-full gap-4">
			<div className="flex items-center gap-3">
				<Button variant="ghost" size="icon" asChild>
					<Link to="/jobs">
						<ArrowLeft className="size-4" />
					</Link>
				</Button>
				<div className="flex-1 min-w-0">
					<div className="flex items-center gap-3">
						<h1 className="font-mono text-sm truncate">{job.id}</h1>
						<StatusBadge status={job.status} />
					</div>
					<div className="flex items-center gap-4 mt-1 text-xs text-muted-foreground">
						<span>
							Created{" "}
							{formatDistanceToNow(new Date(job.created_at), {
								addSuffix: true,
							})}
						</span>
						{job.agent_id && (
							<span>Agent: <span className="font-mono">{job.agent_id.slice(0, 8)}</span></span>
						)}
						{duration != null && <span>Duration: {duration}s</span>}
						{job.failure_reason && (
							<span className="text-destructive">{job.failure_reason}</span>
						)}
					</div>
				</div>
				{!isTerminal(job.status) && (
					<Button
						variant="outline"
						size="sm"
						onClick={handleCancel}
						disabled={cancelJob.isPending}
						className="text-destructive border-destructive/30 hover:bg-destructive/10"
					>
						<XCircle className="size-4 mr-1" />
						Cancel
					</Button>
				)}
			</div>

			<div className="flex-1 min-h-0 rounded-md border bg-black/40 overflow-hidden">
				{logsLoading ? (
					<div className="flex items-center justify-center h-full text-muted-foreground text-sm">
						Loading logs…
					</div>
				) : (
					<LogViewer lines={logs ?? []} autoScroll={!isTerminal(job.status)} />
				)}
			</div>
		</div>
	);
}
