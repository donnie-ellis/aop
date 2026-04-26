import { ScrollArea } from "@/components/ui/scroll-area";
import type { JobLogLine } from "@/types";
import Ansi from "ansi-to-react";
import { useEffect, useRef } from "react";

interface Props {
	lines: JobLogLine[];
	autoScroll?: boolean;
}

export function LogViewer({ lines, autoScroll = true }: Props) {
	const bottomRef = useRef<HTMLDivElement>(null);

	useEffect(() => {
		if (autoScroll) {
			bottomRef.current?.scrollIntoView({ behavior: "smooth" });
		}
	}, [lines, autoScroll]);

	if (lines.length === 0) {
		return (
			<div className="flex items-center justify-center h-full text-muted-foreground font-mono text-sm">
				No output yet.
			</div>
		);
	}

	return (
		<ScrollArea className="h-full w-full">
			<div className="font-mono text-xs leading-5 p-4 space-y-0">
				{lines.map((l) => (
					<div
						key={l.seq}
						className={l.stream === "stderr" ? "text-red-400" : ""}
					>
						<Ansi>{l.line}</Ansi>
					</div>
				))}
				<div ref={bottomRef} />
			</div>
		</ScrollArea>
	);
}
