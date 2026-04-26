import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
	Dialog,
	DialogContent,
	DialogHeader,
	DialogTitle,
	DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { useCreateJob } from "@/hooks/useJobs";
import { useProjects } from "@/hooks/useProjects";
import { useCreateTemplate, useDeleteTemplate, useTemplates } from "@/hooks/useTemplates";
import type { JobTemplate } from "@/types";
import { useForm } from "@tanstack/react-form";
import { formatDistanceToNow } from "date-fns";
import { Play, Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { z } from "zod";

const createSchema = z.object({
	name: z.string().min(1, "Name is required"),
	project_id: z.string().min(1, "Project is required"),
	playbook: z.string().min(1, "Playbook path is required"),
	description: z.string(),
});

function CreateTemplateDialog({ onClose }: { onClose: () => void }) {
	const { data: projects } = useProjects();
	const createTemplate = useCreateTemplate();

	const form = useForm({
		defaultValues: {
			name: "",
			project_id: "",
			playbook: "",
			description: "",
		},
		onSubmit: async ({ value }) => {
			try {
				await createTemplate.mutateAsync(value);
				toast.success("Template created");
				onClose();
			} catch {
				toast.error("Failed to create template");
			}
		},
	});

	return (
		<form
			onSubmit={(e) => {
				e.preventDefault();
				form.handleSubmit();
			}}
			className="space-y-4 pt-2"
		>
			<form.Field
				name="name"
				validators={{
					onBlur: ({ value }) => {
						const r = createSchema.shape.name.safeParse(value);
						if (!r.success) return r.error.issues[0]?.message;
					},
				}}
			>
				{(field) => (
					<div className="space-y-1">
						<Label>Name</Label>
						<Input
							value={field.state.value}
							onChange={(e) => field.handleChange(e.target.value)}
							onBlur={field.handleBlur}
							placeholder="Deploy web app"
						/>
						{field.state.meta.errors[0] && (
							<p className="text-xs text-destructive">{String(field.state.meta.errors[0])}</p>
						)}
					</div>
				)}
			</form.Field>

			<form.Field
				name="project_id"
				validators={{
					onBlur: ({ value }) => {
						const r = createSchema.shape.project_id.safeParse(value);
						if (!r.success) return r.error.issues[0]?.message;
					},
				}}
			>
				{(field) => (
					<div className="space-y-1">
						<Label>Project</Label>
						<Select
							value={field.state.value}
							onValueChange={field.handleChange}
						>
							<SelectTrigger>
								<SelectValue placeholder="Select a project" />
							</SelectTrigger>
							<SelectContent>
								{projects?.map((p) => (
									<SelectItem key={p.id} value={p.id}>
										{p.name}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
						{field.state.meta.errors[0] && (
							<p className="text-xs text-destructive">{String(field.state.meta.errors[0])}</p>
						)}
					</div>
				)}
			</form.Field>

			<form.Field
				name="playbook"
				validators={{
					onBlur: ({ value }) => {
						const r = createSchema.shape.playbook.safeParse(value);
						if (!r.success) return r.error.issues[0]?.message;
					},
				}}
			>
				{(field) => (
					<div className="space-y-1">
						<Label>Playbook</Label>
						<Input
							value={field.state.value}
							onChange={(e) => field.handleChange(e.target.value)}
							onBlur={field.handleBlur}
							placeholder="site.yaml"
							className="font-mono text-sm"
						/>
						{field.state.meta.errors[0] && (
							<p className="text-xs text-destructive">{String(field.state.meta.errors[0])}</p>
						)}
					</div>
				)}
			</form.Field>

			<form.Field name="description">
				{(field) => (
					<div className="space-y-1">
						<Label>Description <span className="text-muted-foreground">(optional)</span></Label>
						<Input
							value={field.state.value}
							onChange={(e) => field.handleChange(e.target.value)}
						/>
					</div>
				)}
			</form.Field>

			<div className="flex justify-end gap-2 pt-2">
				<Button type="button" variant="outline" onClick={onClose}>
					Cancel
				</Button>
				<form.Subscribe selector={(s) => s.isSubmitting}>
					{(isSubmitting) => (
						<Button type="submit" disabled={isSubmitting}>
							{isSubmitting ? "Creating…" : "Create"}
						</Button>
					)}
				</form.Subscribe>
			</div>
		</form>
	);
}

function RunDialog({
	template,
	onClose,
}: { template: JobTemplate; onClose: () => void }) {
	const createJob = useCreateJob();
	const navigate = useNavigate();

	const handleRun = async () => {
		try {
			const job = await createJob.mutateAsync({ templateId: template.id });
			toast.success("Job dispatched");
			onClose();
			navigate(`/jobs/${job.id}`);
		} catch {
			toast.error("Failed to dispatch job");
		}
	};

	return (
		<div className="space-y-4 pt-2">
			<p className="text-sm text-muted-foreground">
				Run <span className="font-semibold text-foreground">{template.name}</span>?
			</p>
			<p className="font-mono text-xs text-muted-foreground">
				Playbook: {template.playbook}
			</p>
			<div className="flex justify-end gap-2">
				<Button variant="outline" onClick={onClose}>
					Cancel
				</Button>
				<Button onClick={handleRun} disabled={createJob.isPending}>
					{createJob.isPending ? "Dispatching…" : "Run now"}
				</Button>
			</div>
		</div>
	);
}

export function TemplatesPage() {
	const { data: templates, isLoading } = useTemplates();
	const deleteTemplate = useDeleteTemplate();
	const [createOpen, setCreateOpen] = useState(false);
	const [runTarget, setRunTarget] = useState<JobTemplate | null>(null);

	const handleDelete = async (id: string) => {
		try {
			await deleteTemplate.mutateAsync(id);
			toast.success("Template deleted");
		} catch {
			toast.error("Failed to delete template");
		}
	};

	return (
		<div className="space-y-4">
			<div className="flex items-center justify-between">
				<h1 className="text-2xl font-semibold">Job Templates</h1>
				<Dialog open={createOpen} onOpenChange={setCreateOpen}>
					<DialogTrigger asChild>
						<Button size="sm">
							<Plus className="size-4 mr-1" />
							New Template
						</Button>
					</DialogTrigger>
					<DialogContent>
						<DialogHeader>
							<DialogTitle>Create Job Template</DialogTitle>
						</DialogHeader>
						<CreateTemplateDialog onClose={() => setCreateOpen(false)} />
					</DialogContent>
				</Dialog>
			</div>

			<Dialog open={!!runTarget} onOpenChange={(o) => !o && setRunTarget(null)}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Run Job</DialogTitle>
					</DialogHeader>
					{runTarget && (
						<RunDialog template={runTarget} onClose={() => setRunTarget(null)} />
					)}
				</DialogContent>
			</Dialog>

			<Card>
				<CardContent className="p-0">
					{isLoading ? (
						<div className="space-y-2 p-6">
							{[...Array(4)].map((_, i) => (
								// biome-ignore lint/suspicious/noArrayIndexKey: skeleton
								<Skeleton key={i} className="h-10 w-full" />
							))}
						</div>
					) : !templates?.length ? (
						<p className="py-12 text-center text-sm text-muted-foreground">
							No templates yet.
						</p>
					) : (
						<table className="w-full text-sm">
							<thead>
								<tr className="border-b text-left text-muted-foreground">
									<th className="px-6 py-3 font-medium">Name</th>
									<th className="px-6 py-3 font-medium">Playbook</th>
									<th className="px-6 py-3 font-medium">Created</th>
									<th className="px-6 py-3" />
								</tr>
							</thead>
							<tbody>
								{templates.map((t) => (
									<tr key={t.id} className="border-b last:border-0 hover:bg-muted/50">
										<td className="px-6 py-3 font-medium">{t.name}</td>
										<td className="px-6 py-3 font-mono text-xs text-muted-foreground">
											{t.playbook}
										</td>
										<td className="px-6 py-3 text-xs text-muted-foreground">
											{formatDistanceToNow(new Date(t.created_at), {
												addSuffix: true,
											})}
										</td>
										<td className="px-6 py-3">
											<div className="flex items-center justify-end gap-1">
												<Button
													variant="ghost"
													size="icon"
													onClick={() => setRunTarget(t)}
												>
													<Play className="size-4" />
												</Button>
												<Button
													variant="ghost"
													size="icon"
													onClick={() => handleDelete(t.id)}
													className="text-destructive hover:text-destructive"
												>
													<Trash2 className="size-4" />
												</Button>
											</div>
										</td>
									</tr>
								))}
							</tbody>
						</table>
					)}
				</CardContent>
			</Card>
		</div>
	);
}
