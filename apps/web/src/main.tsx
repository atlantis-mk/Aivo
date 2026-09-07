import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { Button } from "@aivo/ui/components/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@aivo/ui/components/card";
import { Input } from "@aivo/ui/components/input";
import "@aivo/ui/styles.css";

function App() {
  return (
    <main className="grid min-h-screen place-items-center bg-background p-6 text-foreground">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Aivo Web</CardTitle>
          <CardDescription>浏览器入口已与 Electron 应用分开管理。</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4">
          <Input placeholder="输入项目名称" />
          <Button>开始使用</Button>
        </CardContent>
      </Card>
    </main>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
