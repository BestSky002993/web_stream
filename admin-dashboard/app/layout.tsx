import type { Metadata } from 'next';
import { Toaster } from 'sonner';
import './globals.css';

export const metadata: Metadata = {
	title: 'Video Storage Admin',
	description: 'Admin dashboard for the video-storage-api backend.',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
	return (
		<html lang="en" suppressHydrationWarning>
			<body className="min-h-screen bg-background text-foreground antialiased">
				{children}
				<Toaster position="top-right" richColors closeButton />
			</body>
		</html>
	);
}
