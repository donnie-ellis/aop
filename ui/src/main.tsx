import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { Toaster } from "@/components/ui/sonner";
import { AuthProvider } from "@/contexts/AuthContext";
import { App } from "./App";
import "./index.css";

document.documentElement.classList.add("dark");

const queryClient = new QueryClient({
	defaultOptions: {
		queries: {
			staleTime: 5000,
			retry: 1,
		},
	},
});

createRoot(document.getElementById("root")!).render(
	<StrictMode>
		<QueryClientProvider client={queryClient}>
			<AuthProvider>
				<App />
				<Toaster />
			</AuthProvider>
		</QueryClientProvider>
	</StrictMode>,
);
