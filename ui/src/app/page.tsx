"use client";

import { useState, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { 
  ShieldCheck, 
  Lock, 
  Search, 
  Waves, 
  Cpu, 
  ClipboardCheck, 
  Key, 
  ShieldAlert,
  ArrowDown,
  Send
} from "lucide-react";

interface Step {
  id: string;
  name: string;
  info: string;
  status: string;
}

const moduleIcons: Record<string, any> = {
  "sentinel-in": ShieldCheck,
  "vault-p1": Lock,
  "navigator": Search,
  "anchor-in": Waves,
  "llm": Cpu,
  "anchor-out": ClipboardCheck,
  "vault-p2": Key,
  "sentinel-out": ShieldAlert,
};

export default function Home() {
  const [prompt, setPrompt] = useState("");
  const [isProcessing, setIsProcessing] = useState(false);
  const [currentStepIndex, setCurrentStepIndex] = useState(-1);
  const [steps, setSteps] = useState<Step[]>([]);
  const [finalOutput, setFinalOutput] = useState("");

  const handleProcess = async () => {
    if (!prompt) return;
    
    setIsProcessing(true);
    setCurrentStepIndex(-1);
    setFinalOutput("");
    
    try {
      const res = await fetch("http://localhost:8090/api/process", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ prompt }),
      });
      const data = await res.json();
      setSteps(data.steps);
      
      // Start sequential animation
      for (let i = 0; i < data.steps.length; i++) {
        setCurrentStepIndex(i);
        await new Promise((resolve) => setTimeout(resolve, 1500)); // Delay per module
      }
      
      setCurrentStepIndex(data.steps.length); // Final state
      setFinalOutput(data.final_output);
    } catch (err) {
      console.error("Failed to process:", err);
    } finally {
      setIsProcessing(false);
    }
  };

  return (
    <main className="min-h-screen p-8 flex flex-col items-center bg-background text-foreground">
      <header className="mb-12 text-center">
        <h1 className="text-4xl font-bold mb-2 glow-text tracking-tight">Bastion RAG</h1>
        <p className="text-muted-foreground opacity-70">Security Governance Framework Demo</p>
      </header>

      {/* Input Section */}
      <div className="w-full max-w-2xl mb-16 relative">
        <input
          type="text"
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          placeholder="Enter a prompt (e.g., 'Check John Doe's balance')"
          className="w-full bg-neutral-900 border border-neutral-800 rounded-lg py-4 px-6 pr-16 focus:outline-none focus:border-accent transition-colors shadow-xl"
          onKeyDown={(e) => e.key === "Enter" && handleProcess()}
          disabled={isProcessing}
        />
        <button 
          onClick={handleProcess}
          disabled={isProcessing || !prompt}
          className="absolute right-2 top-2 p-3 rounded-md bg-accent text-black hover:opacity-80 transition-opacity disabled:opacity-50"
        >
          <Send size={20} />
        </button>
      </div>

      {/* Pipeline Visualization */}
      <div className="flex flex-col items-center gap-0 w-full max-w-4xl">
        {steps.map((step, index) => {
          const Icon = moduleIcons[step.id] || ShieldCheck;
          const isActive = index === currentStepIndex;
          const isCompleted = index < currentStepIndex;
          
          return (
            <div key={step.id} className="flex flex-col items-center w-full">
              <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ 
                  opacity: 1, 
                  y: 0,
                  scale: isActive ? 1.05 : 1,
                }}
                className={`relative flex items-center gap-6 p-6 rounded-xl border w-full max-w-xl transition-all duration-500 ${
                  isActive 
                    ? "border-accent glow-border bg-accent/5 animate-glow" 
                    : isCompleted 
                      ? "border-accent/30 bg-accent/5 opacity-80" 
                      : "border-neutral-800 bg-neutral-900/50 opacity-40"
                }`}
              >
                <div className={`p-4 rounded-lg ${isActive || isCompleted ? "bg-accent text-black" : "bg-neutral-800 text-neutral-400"}`}>
                  <Icon size={32} />
                </div>
                
                <div className="flex-1">
                  <h3 className={`text-xl font-semibold mb-1 ${isActive ? "text-accent" : "text-neutral-300"}`}>
                    {step.name}
                  </h3>
                  <AnimatePresence mode="wait">
                    {(isActive || isCompleted) && (
                      <motion.p
                        initial={{ opacity: 0, height: 0 }}
                        animate={{ opacity: 1, height: "auto" }}
                        className="text-sm text-neutral-400 leading-relaxed"
                      >
                        {step.info}
                      </motion.p>
                    )}
                  </AnimatePresence>
                </div>

                {isCompleted && (
                  <motion.div 
                    initial={{ scale: 0 }}
                    animate={{ scale: 1 }}
                    className="text-accent absolute -right-2 -top-2 bg-background p-1 rounded-full border border-accent"
                  >
                    <ClipboardCheck size={16} />
                  </motion.div>
                )}
              </motion.div>

              {index < steps.length - 1 && (
                <div className="pipeline-line opacity-20" />
              )}
            </div>
          );
        })}

        {/* Final Output */}
        <AnimatePresence>
          {finalOutput && (
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              className="mt-12 p-8 rounded-2xl border-2 border-accent/50 bg-neutral-900/80 w-full max-w-2xl text-center shadow-2xl"
            >
              <h2 className="text-xs uppercase tracking-widest text-accent mb-4 opacity-70">Final System Output</h2>
              <p className="text-2xl font-medium leading-relaxed italic">
                "{finalOutput}"
              </p>
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      <footer className="mt-20 py-8 opacity-30 text-xs">
        © 2026 Bastion Security Framework | Confidential Demo
      </footer>
    </main>
  );
}
