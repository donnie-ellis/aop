import { Badge } from "@/components/ui/badge";
import type { JobStatus } from "@/types";
import { cn } from "@/lib/utils";

const config: Record<
	JobStatus | "online" | "offline",
	{ label: string; className: string }
> = {
	pending: {
		label: "Pending",
		className: "bg-muted text-muted-foreground border-muted-foreground/30",
	},
	dispatched: {
		label: "Dispatched",
		className: "bg-blue-500/10 text-blue-400 border-blue-500/30",
	},
	running: {
		label: "Running",
		className: "bg-yellow-500/10 text-yellow-400 border-yellow-500/30",
	},
	success: {
		label: "Success",
		className: "bg-green-500/10 text-green-400 border-green-500/30",
	},
	failed: {
		label: "Failed",
		className: "bg-red-500/10 text-red-400 border-red-500/30",
	},
	cancelled: {
		label: "Cancelled",
		className: "bg-muted text-muted-foreground border-muted-foreground/30",
	},
	online: {
		label: "Online",
		className: "bg-green-500/10 text-green-400 border-green-500/30",
	},
	offline: {
		label: "Offline",
		className: "bg-muted text-muted-foreground border-muted-foreground/30",
	},
};

interface Props {
	status: JobStatus | "online" | "offline";
	className?: string;
}

export function StatusBadge({ status, className }: Props) {
	const { label, className: statusClass } = config[status] ?? config.pending;
	return (
		<Badge
			variant="outline"
			className={cn("font-mono text-xs", statusClass, className)}
		>
			{label}
		</Badge>
	);
}
