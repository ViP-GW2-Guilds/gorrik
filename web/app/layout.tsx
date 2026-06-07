import type { Metadata } from "next";
import { Geist } from "next/font/google";
import "./globals.css";

const geist = Geist({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: "Gorrik",
  description: "GW2 combat log archive",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className="dark h-full">
      <body className={`${geist.className} bg-background text-foreground antialiased h-full`}>
        {children}
      </body>
    </html>
  );
}
