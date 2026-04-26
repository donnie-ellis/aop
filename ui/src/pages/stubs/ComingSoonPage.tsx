import { Construction } from "lucide-react";

export function ComingSoonPage({ title }: { title: string }) {
	return (
		<div className="flex flex-col items-center justify-center h-full gap-3 text-muted-foreground py-32">
			<Construction className="size-10" />
			<h1 className="text-2xl font-semibold text-foreground">{title}</h1>
			<p className="text-sm">Coming in a future release.</p>
		</div>
	);
}
