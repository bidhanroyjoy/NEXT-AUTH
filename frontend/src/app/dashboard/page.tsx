"use client";

import { useSession, signOut } from "next-auth/react";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import Link from "next/link";

export default function Dashboard() {
  const { data: session, status } = useSession();
  const router = useRouter();

  useEffect(() => {
    if (status === "unauthenticated") {
      router.push("/login");
    }
  }, [status, router]);

  if (status === "loading") {
    return <p className="text-center mt-10">Loading...</p>;
  }

  return (
    <div className="min-h-screen bg-gray-50 flex flex-col pt-12 items-center">
      <div className="bg-white p-8 rounded-lg shadow-md w-full max-w-2xl text-center border border-gray-200">
        <h1 className="text-3xl font-extrabold text-gray-900 mb-4">Welcome to your Dashboard</h1>
        <p className="text-lg text-gray-700 mb-8">
          You are successfully logged in as <span className="font-semibold text-indigo-600">{session?.user?.email}</span>.
        </p>
        
        <div className="bg-gray-50 p-6 rounded-md border border-gray-200 mb-8 text-left">
          <h3 className="text-lg font-bold text-gray-800 mb-2">Security Hub</h3>
          <p className="text-sm text-gray-600 mb-4">Manage your account security settings here.</p>
          <Link
            href="/setup-2fa"
            className="w-full inline-flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-green-600 hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500"
          >
            🔐 Configure Google Authenticator (2FA)
          </Link>
        </div>

        <button
          onClick={() => signOut({ callbackUrl: "/login" })}
          className="w-full inline-flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
        >
          Sign Out
        </button>
      </div>
    </div>
  );
}
