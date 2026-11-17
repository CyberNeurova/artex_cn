"use client";

import { useEffect, useState } from "react";

import { Loader2 } from "lucide-react";

import { auth } from "@/lib/auth";

export default function OAuthCallbackPage() {
  const [error, setError] = useState("");

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const token = params.get("token");
    const err = params.get("error");
    if (token) {
      auth.setToken(token);
      window.location.replace("/function/tasks");
    } else {
      setError(err ? decodeURIComponent(err) : "Authentication failed: no token received");
    }
  }, []);

  if (error) {
    return (
      <div className="flex min-h-dvh items-center justify-center p-8">
        <div className="space-y-4 text-center">
          <p className="text-sm text-destructive">{error}</p>
          <a href="/login" className="text-sm text-primary underline-offset-4 hover:underline">
            Back to sign in
          </a>
        </div>
      </div>
    );
  }

  return (
    <div role="status" className="flex min-h-dvh items-center justify-center gap-2 text-muted-foreground">
      <Loader2 className="size-4 animate-spin" />
      Signing in…
    </div>
  );
}
