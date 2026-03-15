"use client";

import { useSession, signOut } from "next-auth/react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export default function Home() {
  const { data: session, status } = useSession();
  const router = useRouter();

  // If loading session, show a spinner or loading text
  if (status === "loading") {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <p className="text-gray-500">Loading session...</p>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 flex flex-col justify-center items-center py-12 sm:px-6 lg:px-8">
      <div className="sm:mx-auto sm:w-full sm:max-w-md text-center">
        <h1 className="text-4xl font-extrabold text-gray-900 mb-4">
          Auth Application
        </h1>
        <p className="text-gray-600 mb-8">
          Welcome to the Next.js & Golang authentication demo.
        </p>

        {status === "authenticated" ? (
           <div className="bg-white py-8 px-4 shadow sm:rounded-lg sm:px-10">
              <h2 className="text-2xl font-bold text-gray-800 mb-4">Dashboard</h2>
              <p className="text-gray-600 mb-6">
                 Logged in as: <span className="font-semibold">{session.user?.email}</span>
              </p>
              <button
                onClick={() => signOut({ callbackUrl: '/login' })}
                className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
              >
                Sign out
              </button>
           </div>
        ) : (
          <div className="space-y-4">
            <Link 
              href="/login" 
              className="w-full flex justify-center py-3 px-4 border border-transparent rounded-md shadow-sm text-md font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
            >
              Go to Login Page
            </Link>
            <Link 
              href="/register" 
              className="w-full flex justify-center py-3 px-4 border border-gray-300 rounded-md shadow-sm text-md font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
            >
              Register a New Account
            </Link>
          </div>
        )}
      </div>
    </div>
  );
}
