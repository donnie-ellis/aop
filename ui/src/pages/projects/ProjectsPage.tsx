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
import { Skeleton } from "@/components/ui/skeleton";
import { useCreateProject, useDeleteProject, useProjects, useSyncProject } from "@/hooks/useProjects";
import { useForm } from "@tanstack/react-form";
import { formatDistanceToNow } from "date-fns";
import { Plus, RefreshCw, Trash2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { z } from "zod";

const createSchema = z.object({
	name: z.string().min(1, "Name is required"),
	repo_url: z.string().url("Must be a valid URL"),
	branch: z.string(),
	inventory_path: z.string().min(1, "Inventory path is required"),
});

function CreateProjectDialog({ onClose }: { onClose: () => void }) {
	const createProject = useCreateProject();

	const form = useForm({
		defaultValues: {
			name: "",
			repo_url: "",
			branch: "main",
			inventory_path: "inventory.ini",
		},
		onSubmit: async ({ value }) => {
			try {
				await createProject.mutateAsync(value);
				toast.success("Project created");
				onClose();
			} catch {
				toast.error("Failed to create project");
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
			{(
				[
					["name", "Name", "text", "my-infra"],
					["repo_url", "Repository URL", "url", "https://github.com/org/repo"],
					["branch", "Branch", "text", "main"],
					["inventory_path", "Inventory Path", "text", "inventory.ini"],
				] as const
			).map(([name, label, type, placeholder]) => (
				<form.Field
					key={name}
					name={name}
					validators={{
						onBlur: ({ value }) => {
							const shape = createSchema.shape[name as keyof typeof createSchema.shape];
							const r = shape.safeParse(value);
							if (!r.success) return r.error.issues[0]?.message;
						},
					}}
				>
					{(field) => (
						<div className="space-y-1">
							<Label>{label}</Label>
							<Input
								type={type}
								value={field.state.value}
								onChange={(e) => field.handleChange(e.target.value)}
								onBlur={field.handleBlur}
								placeholder={placeholder}
								className={name === "repo_url" || name === "inventory_path" ? "font-mono text-sm" : ""}
							/>
							{field.state.meta.errors[0] && (
								<p className="text-xs text-destructive">{String(field.state.meta.errors[0])}</p>
							)}
						</div>
					)}
				</form.Field>
			))}

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

export function ProjectsPage() {
	const { data: projects, isLoading } = useProjects();
	const deleteProject = useDeleteProject();
	const syncProject = useSyncProject();
	const [createOpen, setCreateOpen] = useState(false);

	const handleDelete = async (id: string) => {
		try {
			await deleteProject.mutateAsync(id);
			toast.success("Project deleted");
		} catch {
			toast.error("Failed to delete project");
		}
	};

	const handleSync = async (id: string) => {
		try {
			await syncProject.mutateAsync(id);
			toast.success("Sync requested");
		} catch {
			toast.error("Failed to request sync");
		}
	};

	return (
		<div className="space-y-4">
			<div className="flex items-center justify-between">
				<h1 className="text-2xl font-semibold">Projects</h1>
				<Dialog open={createOpen} onOpenChange={setCreateOpen}>
					<DialogTrigger asChild>
						<Button size="sm">
							<Plus className="size-4 mr-1" />
							New Project
						</Button>
					</DialogTrigger>
					<DialogContent>
						<DialogHeader>
							<DialogTitle>Create Project</DialogTitle>
						</DialogHeader>
						<CreateProjectDialog onClose={() => setCreateOpen(false)} />
					</DialogContent>
				</Dialog>
			</div>

			<Card>
				<CardContent className="p-0">
					{isLoading ? (
						<div className="space-y-2 p-6">
							{[...Array(4)].map((_, i) => (
								// biome-ignore lint/suspicious/noArrayIndexKey: skeleton
								<Skeleton key={i} className="h-10 w-full" />
							))}
						</div>
					) : !projects?.length ? (
						<p className="py-12 text-center text-sm text-muted-foreground">
							No projects yet.
						</p>
					) : (
						<table className="w-full text-sm">
							<thead>
								<tr className="border-b text-left text-muted-foreground">
									<th className="px-6 py-3 font-medium">Name</th>
									<th className="px-6 py-3 font-medium">Repository</th>
									<th className="px-6 py-3 font-medium">Branch</th>
									<th className="px-6 py-3 font-medium">Last Synced</th>
									<th className="px-6 py-3" />
								</tr>
							</thead>
							<tbody>
								{projects.map((p) => (
									<tr key={p.id} className="border-b last:border-0 hover:bg-muted/50">
										<td className="px-6 py-3 font-medium">{p.name}</td>
										<td className="px-6 py-3 font-mono text-xs text-muted-foreground max-w-xs truncate">
											{p.repo_url}
										</td>
										<td className="px-6 py-3 font-mono text-xs">
											{p.branch}
										</td>
										<td className="px-6 py-3 text-xs text-muted-foreground">
											{p.last_synced_at
												? formatDistanceToNow(new Date(p.last_synced_at), {
														addSuffix: true,
													})
												: "Never"}
										</td>
										<td className="px-6 py-3">
											<div className="flex items-center justify-end gap-1">
												<Button
													variant="ghost"
													size="icon"
													onClick={() => handleSync(p.id)}
													disabled={syncProject.isPending}
												>
													<RefreshCw className="size-4" />
												</Button>
												<Button
													variant="ghost"
													size="icon"
													onClick={() => handleDelete(p.id)}
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
