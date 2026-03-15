"use client";

import { useState, useEffect } from "react";
import { useSession } from "next-auth/react";
import { useRouter } from "next/navigation";
import api from "@/lib/axios";
import { QRCodeSVG } from "qrcode.react";
import Link from "next/link";

export default function Setup2FAPage() {
  const { data: session, status } = useSession();
  const router = useRouter();

  const [step, setStep] = useState<"loading" | "status" | "setup" | "verify" | "enabled">("loading");
  const [otpauthURL, setOtpauthURL] = useState("");
  const [secret, setSecret] = useState("");
  const [code, setCode] = useState("");
  const [is2FAEnabled, setIs2FAEnabled] = useState(false);
  const [msg, setMsg] = useState("");
  const [isError, setIsError] = useState(false);
  const [disableCode, setDisableCode] = useState("");

  // Redirect unauthenticated users
  useEffect(() => {
    if (status === "unauthenticated") {
      router.push("/login");
    }
  }, [status, router]);

  // Fetch current 2FA status
  useEffect(() => {
    if (status !== "authenticated") return;

    const fetch2FAStatus = async () => {
      try {
        const res = await api.get("/auth/2fa/status");
        setIs2FAEnabled(res.data.is_2fa_enabled);
        setStep("status");
      } catch {
        setStep("status");
      }
    };
    fetch2FAStatus();
  }, [status]);

  const handleSetup = async () => {
    setIsError(false);
    setMsg("");
    try {
      const res = await api.post("/auth/2fa/setup");
      setOtpauthURL(res.data.otpauth_url);
      setSecret(res.data.secret);
      setStep("setup");
    } catch (err: any) {
      setIsError(true);
      setMsg(err.response?.data?.error || "Failed to setup 2FA");
    }
  };

  const handleVerifySetup = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsError(false);
    setMsg("");
    try {
      await api.post("/auth/2fa/verify-setup", { code });
      setIs2FAEnabled(true);
      setStep("enabled");
      setMsg("2FA has been enabled successfully!");
    } catch (err: any) {
      setIsError(true);
      setMsg(err.response?.data?.error || "Invalid code");
    }
  };

  const handleDisable = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsError(false);
    setMsg("");
    try {
      await api.post("/auth/2fa/disable", { code: disableCode });
      setIs2FAEnabled(false);
      setStep("status");
      setMsg("2FA has been disabled successfully.");
      setDisableCode("");
    } catch (err: any) {
      setIsError(true);
      setMsg(err.response?.data?.error || "Failed to disable 2FA");
    }
  };

  if (status === "loading" || step === "loading") {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <p className="text-gray-500">Loading...</p>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
      <div className="sm:mx-auto sm:w-full sm:max-w-md">
        <h2 className="mt-6 text-center text-3xl font-extrabold text-gray-900">
          Two-Factor Authentication
        </h2>
        <p className="mt-2 text-center text-sm text-gray-600">
          Secure your account with Google Authenticator
        </p>
      </div>

      <div className="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
        <div className="bg-white py-8 px-4 shadow sm:rounded-lg sm:px-10">
          {msg && (
            <div className={`text-sm text-center font-bold mb-4 ${isError ? "text-red-500" : "text-green-500"}`}>
              {msg}
            </div>
          )}

          {/* Status View - Show current 2FA state */}
          {step === "status" && !is2FAEnabled && (
            <div className="text-center space-y-4">
              <div className="mx-auto w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center text-3xl">
                🔓
              </div>
              <p className="text-gray-700">Two-factor authentication is <strong className="text-red-500">not enabled</strong>.</p>
              <p className="text-sm text-gray-500">
                Add an extra layer of security to your account by enabling Google Authenticator.
              </p>
              <button
                onClick={handleSetup}
                className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
              >
                Enable 2FA
              </button>
            </div>
          )}

          {step === "status" && is2FAEnabled && (
            <div className="space-y-4">
              <div className="text-center">
                <div className="mx-auto w-16 h-16 bg-green-100 rounded-full flex items-center justify-center text-3xl">
                  🔒
                </div>
                <p className="mt-3 text-gray-700">Two-factor authentication is <strong className="text-green-600">enabled</strong>.</p>
              </div>

              <hr className="my-4" />

              <p className="text-sm text-gray-600 text-center">To disable 2FA, enter your current authenticator code:</p>
              <form onSubmit={handleDisable} className="space-y-4">
                <input
                  type="text"
                  required
                  maxLength={6}
                  placeholder="Enter 6-digit code"
                  value={disableCode}
                  onChange={(e) => setDisableCode(e.target.value.replace(/\D/g, ""))}
                  className="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm placeholder-gray-400 focus:outline-none focus:ring-red-500 focus:border-red-500 sm:text-sm text-gray-900 text-center tracking-widest text-lg"
                />
                <button
                  type="submit"
                  className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
                >
                  Disable 2FA
                </button>
              </form>
            </div>
          )}

          {/* Setup View - Show QR Code */}
          {step === "setup" && (
            <div className="space-y-6">
              <div className="text-center">
                <p className="text-sm text-gray-600 mb-4">
                  Scan this QR code with <strong>Google Authenticator</strong>:
                </p>
                <div className="flex justify-center p-4 bg-white border-2 border-gray-200 rounded-lg inline-block mx-auto">
                  <QRCodeSVG value={otpauthURL} size={200} />
                </div>
              </div>

              <div className="bg-gray-50 rounded-md p-3">
                <p className="text-xs text-gray-500 text-center mb-1">Or enter this secret manually:</p>
                <p className="text-sm font-mono text-center text-gray-800 break-all select-all">{secret}</p>
              </div>

              <form onSubmit={handleVerifySetup} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700">Enter the 6-digit code from your app</label>
                  <input
                    type="text"
                    required
                    maxLength={6}
                    placeholder="000000"
                    value={code}
                    onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
                    className="mt-1 appearance-none block w-full px-3 py-2 border border-indigo-300 rounded-md shadow-sm placeholder-gray-400 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm text-gray-900 text-center tracking-widest text-lg"
                    autoFocus
                  />
                </div>
                <button
                  type="submit"
                  className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none"
                >
                  Verify & Enable 2FA
                </button>
              </form>
            </div>
          )}

          {/* Enabled confirmation */}
          {step === "enabled" && (
            <div className="text-center space-y-4">
              <div className="mx-auto w-16 h-16 bg-green-100 rounded-full flex items-center justify-center text-3xl">
                ✅
              </div>
              <p className="text-gray-700 font-medium">2FA is now active!</p>
              <p className="text-sm text-gray-500">
                You will be asked for a Google Authenticator code every time you log in.
              </p>
              <Link
                href="/login"
                className="w-full inline-flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700"
              >
                Go to Login
              </Link>
            </div>
          )}

          <div className="mt-6 text-center">
            <Link href="/" className="text-sm font-medium text-indigo-600 hover:text-indigo-500">
              ← Back to Home
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
