import { code } from "@streamdown/code";
import { cjk } from "@streamdown/cjk";
import { math } from "@streamdown/math";
import { mermaid } from "@streamdown/mermaid";
import { FileLinkIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import * as echarts from "echarts/core";
import { BarChart, LineChart, PieChart, ScatterChart } from "echarts/charts";
import {
 DatasetComponent,
 GridComponent,
 LegendComponent,
 TitleComponent,
 TooltipComponent,
 TransformComponent,
} from "echarts/components";
import { CanvasRenderer } from "echarts/renderers";
import type { ECharts, EChartsCoreOption } from "echarts/core";
import { useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { toast } from "sonner";
import { Streamdown, type Components, type ControlsConfig, type CustomRendererProps } from "streamdown";

import {
 localPathFromMarkdownHref,
 localPathFromText,
 markdownHrefForLocalPath,
} from "@/components/markdown-local-path";
import { cn } from "@/lib/utils";

type MarkdownStreamFactory = () => AsyncGenerator<string, void, unknown>;

type MarkdownBaseProps = {
 className?: string;
};

type MarkdownContentProps = MarkdownBaseProps & {
 content: string;
 isFinished?: boolean;
 stream?: never;
};

type MarkdownStreamProps = MarkdownBaseProps & {
 content?: never;
 isFinished?: never;
 stream: MarkdownStreamFactory;
};

type MarkdownProps = MarkdownContentProps | MarkdownStreamProps;

type EchartsCodeBlockProps = CustomRendererProps;

const markdownContentResizeEvent = "aivo-markdown-content-resize";

echarts.use([
 BarChart,
 LineChart,
 PieChart,
 ScatterChart,
 GridComponent,
 LegendComponent,
 TitleComponent,
 TooltipComponent,
 DatasetComponent,
 TransformComponent,
 CanvasRenderer,
]);

const streamdownComponents: Components = {
 a: ({ children, className, href: rawHref, node: _node, title: rawTitle, ...props }) => {
 const href = typeof rawHref === "string" ? rawHref : undefined;
 const title = typeof rawTitle === "string" ? rawTitle : undefined;
 const localPath = href ? localPathFromMarkdownHref(href) : undefined;
 if (localPath) {
 return (
 <a
 {...props}
 className={cn(
 "inline-flex max-w-full items-baseline gap-1 align-baseline font-medium text-blue-600 no-underline underline-offset-4 hover:underline focus-visible:rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/45 dark:text-blue-300",
 className,
 )}
 data-local-path-link="true"
 href={href}
 onClick={(event) => {
 event.preventDefault();
 void openLocalPath(localPath);
 }}
 target={undefined}
 title={title ?? `使用系统默认应用打开 ${localPath}`}
 >
 <LocalPathLinkContent>{children}</LocalPathLinkContent>
 </a>
 );
 }

 return (
 <a
 {...props}
 className={cn("wrap-anywhere font-medium text-primary underline", className)}
 data-streamdown="link"
 href={href}
 rel="noopener noreferrer"
 target="_blank"
 title={title}
 >
 {children}
 </a>
 );
 },
 inlineCode: ({ children, className, node: _node }) => {
 const text = typeof children === "string" ? children : undefined;
 const localPath = text
 ? localPathFromText(text, window.aivo?.platform)
 : undefined;
 if (!localPath) {
 return (
 <code
 className={cn("rounded bg-muted px-1.5 py-0.5 font-mono text-sm", className)}
 data-streamdown="inline-code"
 >
 {children}
 </code>
 );
 }

 return (
 <button
 aria-label={`使用系统默认应用打开 ${localPath}`}
 className={cn(
 "inline-flex max-w-full cursor-pointer appearance-none items-baseline gap-1 border-0 bg-transparent p-0 align-baseline font-sans font-medium text-blue-600 underline-offset-4 hover:underline focus-visible:rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/45 dark:text-blue-300",
 className,
 )}
 data-local-path-link="true"
 onClick={() => void openLocalPath(localPath)}
 title={`使用系统默认应用打开 ${localPath}`}
 type="button"
 >
 <LocalPathLinkContent>{children}</LocalPathLinkContent>
 </button>
 );
 },
 img: (props) => {
 const src = typeof props.src === "string" ? props.src : undefined;
 const alt = typeof props.alt === "string" ? props.alt : "";
 const title = typeof props.title === "string" ? props.title : undefined;
 if (!src) return null;

 return <img alt={alt} loading="lazy" src={src} title={title} />;
 },
};

function LocalPathLinkContent({ children }: { children: ReactNode }) {
 return (
 <>
 <HugeiconsIcon
 aria-hidden="true"
 className="relative top-[0.125em] size-[1.05em] shrink-0 self-baseline"
 icon={FileLinkIcon}
 strokeWidth={2}
 />
 <span className="min-w-0 break-all text-left">{children}</span>
 </>
 );
}

const streamdownPlugins = {
 code,
 cjk,
 math,
 mermaid,
 renderers: [{ language: "echarts", component: EchartsCodeBlock }],
};

const streamdownStreamingPlugins = {
 code,
 cjk,
};

const markdownControls = {
 code: true,
 mermaid: true,
 table: true,
} satisfies ControlsConfig;

export function Markdown(props: MarkdownProps) {
 const isStreamSource = typeof props.stream === "function";
 const stream = isStreamSource ? props.stream : undefined;
 const [streamContent, setStreamContent] = useState("");
 const [streamFinished, setStreamFinished] = useState(!isStreamSource);

 useEffect(() => {
 if (!stream) return;

 const streamFactory = stream;
 let isCancelled = false;
 let generator: AsyncGenerator<string, void, unknown> | undefined;

 async function consumeStream() {
 setStreamContent("");
 setStreamFinished(false);

 try {
 generator = streamFactory();
 for await (const chunk of generator) {
 if (isCancelled) return;
 setStreamContent((currentContent) => currentContent + chunk);
 }
 } catch (error) {
 if (!isCancelled) {
 console.error("Markdown stream failed", error);
 }
 } finally {
 if (!isCancelled) setStreamFinished(true);
 }
 }

 void consumeStream();

 return () => {
 isCancelled = true;
 void generator?.return(undefined);
 };
 }, [stream]);

 const content = isStreamSource ? streamContent : props.content;
 const isFinished = isStreamSource ? streamFinished : props.isFinished ?? true;

 return (
 <MarkdownViewer
 className={props.className}
 content={content}
 isFinished={isFinished}
 />
 );
}

function MarkdownViewer({ content, isFinished, className }: MarkdownContentProps) {
 if (!content.trim()) return null;

 return (
 <Streamdown
 animated={false}
 className={cn("aivo-markdown break-words text-sm/relaxed", className)}
 components={streamdownComponents}
 controls={markdownControls}
 isAnimating={!isFinished}
 lineNumbers={false}
 linkSafety={{ enabled: false }}
 mode={isFinished ? "static" : "streaming"}
 plugins={isFinished ? streamdownPlugins : streamdownStreamingPlugins}
 urlTransform={safeMarkdownUrl}
 >
 {content}
 </Streamdown>
 );
}

function EchartsCodeBlock({ code: codeStr, isIncomplete }: EchartsCodeBlockProps) {
 const chartRef = useRef<HTMLDivElement>(null);
 const chartInstanceRef = useRef<ECharts | null>(null);
 const parsedOption = useMemo(() => parseEchartsOption(codeStr), [codeStr]);
 const shouldShowChart = !isIncomplete;

 useLayoutEffect(() => {
 const chartElement = chartRef.current;
 if (!chartElement || !parsedOption.ok || !shouldShowChart) return;

 const chart = echarts.getInstanceByDom(chartElement) ?? echarts.init(chartElement, null, { renderer: "canvas" });
 chartInstanceRef.current = chart;
 chart.setOption(normalizeEchartsOption(parsedOption.option), true);
 const notifyContentResize = () => {
 window.dispatchEvent(new CustomEvent(markdownContentResizeEvent));
 };
 chart.resize();
 notifyContentResize();

 const resizeObserver = new ResizeObserver(() => {
 chart.resize();
 notifyContentResize();
 });
 resizeObserver.observe(chartElement);

 return () => {
 resizeObserver.disconnect();
 chart.dispose();
 chartInstanceRef.current = null;
 };
 }, [parsedOption, shouldShowChart]);

 if (!shouldShowChart) {
 return (
 <div
 className="my-3 overflow-hidden rounded-md border border-border bg-background"
 data-assistant-hover-ignore="true"
 >
 <div className="border-b border-border px-3 py-2 text-xs  text-muted-foreground">ECHART</div>
 <div className="p-4 text-sm text-muted-foreground">解析中...</div>
 </div>
 );
 }

 if (!parsedOption.ok) {
 return (
 <div
 className="my-3 overflow-hidden rounded-md border border-destructive/30 bg-destructive/5"
 data-assistant-hover-ignore="true"
 >
 <div className="border-b border-destructive/20 px-3 py-2 text-xs  text-destructive">ECHART</div>
 <div className="p-4 text-sm text-destructive">ECharts JSON 配置解析失败：{parsedOption.message}</div>
 </div>
 );
 }

 return (
 <div
 className="my-3 overflow-hidden rounded-md border border-border bg-background"
 data-assistant-hover-ignore="true"
 >
 <div className="border-b border-border px-3 py-2 text-xs  text-muted-foreground">ECHART</div>
 <div className="h-[400px] min-w-0" ref={chartRef} />
 </div>
 );
}

function parseEchartsOption(codeStr: string): { ok: true; option: EChartsCoreOption } | { ok: false; message: string } {
 try {
 const parsed = JSON.parse(codeStr) as unknown;
 if (!isRecord(parsed)) {
 return { ok: false, message: "配置必须是 JSON 对象" };
 }

 return { ok: true, option: parsed as EChartsCoreOption };
 } catch (error) {
 return { ok: false, message: error instanceof Error ? error.message : "未知错误" };
 }
}

function normalizeEchartsOption(option: EChartsCoreOption): EChartsCoreOption {
 return {
 ...option,
 tooltip: normalizeEchartsTooltip((option as { tooltip?: unknown }).tooltip),
 };
}

function normalizeEchartsTooltip(tooltip: unknown) {
 if (Array.isArray(tooltip)) {
 return tooltip.map((item) => normalizeEchartsTooltipObject(item));
 }

 return normalizeEchartsTooltipObject(tooltip);
}

function normalizeEchartsTooltipObject(tooltip: unknown) {
 const tooltipOption = isRecord(tooltip) ? tooltip : {};

 return {
 ...tooltipOption,
 appendTo: "body",
 appendToBody: true,
 enterable: false,
 };
}

function isRecord(value: unknown): value is Record<string, unknown> {
 return typeof value === "object" && value !== null && !Array.isArray(value);
}

function safeMarkdownUrl(value: string, key: string) {
 if (!value) return "";

 if (key === "href") {
 const localPathHref = markdownHrefForLocalPath(value, window.aivo?.platform);
 if (localPathHref) return localPathHref;
 }

 try {
 const url = new URL(value, window.location.href);
 if (key === "src") {
 return ["http:", "https:"].includes(url.protocol) ? value : "";
 }

 return ["http:", "https:", "mailto:"].includes(url.protocol) ? value : "";
 } catch {
 return "";
 }
}

async function openLocalPath(target: string) {
 try {
 await window.aivo.openPath(target);
 } catch (error) {
 const detail = error instanceof Error ? error.message : String(error);
 toast.error("无法打开文件地址", { description: detail });
 }
}
