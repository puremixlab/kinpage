import type { DailyActivity } from "./mygo";

export interface WritingActivityView {
  refresh(): Promise<void>;
}

type HeatmapMode = "day" | "week" | "cumulative";

const SVG_NAMESPACE = "http://www.w3.org/2000/svg";
const DAY_MS = 86400000;
const TREND_WIDTH = 1056;
const TREND_HEIGHT = 240;
const TREND_PAD_TOP = 16;
const TREND_PAD_BOTTOM = 10;

const dayKey = (date: Date) => `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
const startOfDay = (date: Date) => new Date(date.getFullYear(), date.getMonth(), date.getDate());
const mondayOf = (date: Date) => {
  const day = startOfDay(date);
  return new Date(day.getFullYear(), day.getMonth(), day.getDate() - (day.getDay() + 6) % 7);
};
const shiftDays = (date: Date, days: number) => new Date(date.getFullYear(), date.getMonth(), date.getDate() + days);
const longDate = (date: Date) => date.toLocaleDateString("zh-CN", { month: "long", day: "numeric" });
const intensity = (count: number) => count <= 0 ? 0 : count === 1 ? 1 : count === 2 ? 2 : count <= 4 ? 3 : 4;
const round = (value: number) => Math.round(value * 100) / 100;

function weekColumns(today: Date, weeks: number) {
  const current = mondayOf(today);
  const columns: Date[][] = [];
  for (let offset = weeks - 1; offset >= 0; offset--) {
    const start = shiftDays(current, -offset * 7);
    columns.push([0, 1, 2, 3, 4, 5, 6].map(day => shiftDays(start, day)));
  }
  return columns;
}

// Catmull-Rom smoothing so the trend line matches the reference curve look.
function smoothPath(points: readonly (readonly [number, number])[]) {
  const first = points[0];
  if (!first) return "";
  let d = `M ${round(first[0])} ${round(first[1])}`;
  for (let i = 1; i < points.length; i++) {
    const previous = points[i - 1] ?? first, current = points[i] ?? first;
    const before = points[i - 2] ?? previous, after = points[i + 1] ?? current;
    const c1x = previous[0] + (current[0] - before[0]) / 6, c1y = previous[1] + (current[1] - before[1]) / 6;
    const c2x = current[0] - (after[0] - previous[0]) / 6, c2y = current[1] - (after[1] - previous[1]) / 6;
    d += ` C ${round(c1x)} ${round(c1y)}, ${round(c2x)} ${round(c2y)}, ${round(current[0])} ${round(current[1])}`;
  }
  return d;
}

export function createWritingActivity(section: HTMLElement, load: () => Promise<DailyActivity[]>): WritingActivityView {
  const doc = section.ownerDocument;
  const $ = <T extends HTMLElement = HTMLElement>(selector: string) => section.querySelector<T>(selector)!;
  const number = new Intl.NumberFormat("zh-CN");
  let revision = 0;
  let data: DailyActivity[] | null = null;
  let mode: HeatmapMode = "day";
  let range = 7;

  const byDay = () => {
    const map = new Map<string, DailyActivity>();
    for (const entry of data ?? []) map.set(entry.date, entry);
    return map;
  };
  const element = (tag: string, className: string) => {
    const node = doc.createElement(tag);
    node.className = className;
    return node;
  };
  const monthLabel = (monthStart: Date | undefined) => {
    const label = element("span", `activity-month${monthStart ? "" : " is-blank"}`);
    label.textContent = monthStart ? `${monthStart.getMonth() + 1}月` : "";
    return label;
  };

  function dailyCell(day: Date, map: Map<string, DailyActivity>) {
    const cell = element("span", "activity-cell");
    const entry = map.get(dayKey(day));
    const count = entry ? entry.articles + entry.drafts : 0;
    cell.dataset.level = String(intensity(count));
    cell.title = `${longDate(day)} · ${count ? `${number.format(count)} 次写作动作` : "无写作动作"}`;
    return cell;
  }

  function renderDayHeatmap(map: Map<string, DailyActivity>) {
    const today = startOfDay(new Date());
    const grid = element("div", "activity-grid");
    for (const week of weekColumns(today, 53)) {
      const column = element("div", "activity-grid-column");
      for (const day of week) {
        if (day > today) column.append(element("span", "activity-cell is-future"));
        else column.append(dailyCell(day, map));
      }
      const monthStart = week.find(day => day.getDate() === 1 && day <= today);
      column.append(monthLabel(monthStart));
      grid.append(column);
    }
    return grid;
  }

  function weekEntry(week: readonly Date[], map: Map<string, DailyActivity>) {
    let articles = 0, drafts = 0;
    for (const day of week) {
      const entry = map.get(dayKey(day));
      if (entry) { articles += entry.articles; drafts += entry.drafts; }
    }
    return { articles, drafts };
  }

  function renderWeekHeatmap(map: Map<string, DailyActivity>, cumulative: boolean) {
    const today = startOfDay(new Date());
    let columns = weekColumns(today, 53);
    if (cumulative) {
      const first = data?.[0]?.date;
      if (first) {
        const earliest = mondayOf(new Date(`${first}T00:00:00`));
        if (earliest < mondayOf(today)) {
          const total = Math.round((mondayOf(today).getTime() - earliest.getTime()) / (7 * DAY_MS)) + 1;
          columns = weekColumns(today, Math.min(260, Math.max(53, total)));
        }
      }
    }
    const grid = element("div", "activity-grid activity-grid-week");
    let cumulativeCount = 0;
    for (const week of columns) {
      const weekStart = week[0] ?? today;
      const column = element("div", "activity-grid-column");
      const { articles, drafts } = weekEntry(week, map);
      const count = articles + drafts;
      cumulativeCount += count;
      const cell = element("span", "activity-cell");
      cell.dataset.level = String(intensity(count));
      const title = `${longDate(weekStart)}当周 · ${count ? `${number.format(count)} 次写作动作` : "无写作动作"}`;
      cell.title = cumulative && count ? `${title} · 截至本周累计 ${number.format(cumulativeCount)} 次` : title;
      column.append(cell);
      const monthStart = week.find(day => day.getDate() === 1 && day <= today);
      column.append(monthLabel(monthStart));
      grid.append(column);
    }
    return grid;
  }

  function renderHeatmap() {
    const container = $("#dashboard-activity-heatmap");
    container.replaceChildren();
    const map = byDay();
    if (!data?.length) {
      const empty = element("p", "dashboard-list-empty");
      empty.textContent = "还没有写作记录，动笔后会在这里看到活动热力图。";
      container.append(empty);
      return;
    }
    container.append(mode === "day" ? renderDayHeatmap(map) : renderWeekHeatmap(map, mode === "cumulative"));
  }

  function renderSummary() {
    const summary = $("#dashboard-activity-summary");
    if (!data?.length) { summary.hidden = true; return; }
    const map = byDay();
    const today = startOfDay(new Date());
    if (mode === "cumulative") {
      const total = data.reduce((sum, entry) => sum + entry.articles + entry.drafts, 0);
      summary.textContent = `全部时间共 ${number.format(total)} 次写作动作 · 覆盖 ${number.format(data.length)} 天`;
    } else {
      const yearTotal = data.filter(entry => entry.date >= dayKey(shiftDays(today, -364))).reduce((sum, entry) => sum + entry.articles + entry.drafts, 0);
      let streak = 0;
      let cursor = map.has(dayKey(today)) ? today : map.has(dayKey(shiftDays(today, -1))) ? shiftDays(today, -1) : null;
      while (cursor && map.has(dayKey(cursor))) { streak++; cursor = shiftDays(cursor, -1); }
      summary.textContent = `近一年 ${number.format(yearTotal)} 次写作动作 · 连续 ${number.format(streak)} 天`;
    }
    summary.hidden = false;
  }

  function renderTrend() {
    const container = $("#dashboard-activity-trend");
    const empty = !data?.length;
    $("#dashboard-activity-range-heading").hidden = empty;
    $("#dashboard-activity-trend-card").hidden = empty;
    if (empty) return;
    container.replaceChildren();
    const map = byDay();
    const today = startOfDay(new Date());
    const series: { date: Date; articles: number; drafts: number }[] = [];
    for (let offset = range - 1; offset >= 0; offset--) {
      const day = shiftDays(today, -offset);
      const entry = map.get(dayKey(day));
      series.push({ date: day, articles: entry?.articles ?? 0, drafts: entry?.drafts ?? 0 });
    }
    const peak = Math.max(...series.map(point => Math.max(point.articles, point.drafts)));
    const yMax = Math.max(4, Math.ceil(peak / 4) * 4);
    const x = (index: number) => (index / Math.max(1, series.length - 1)) * TREND_WIDTH;
    const y = (value: number) => TREND_PAD_TOP + (1 - value / yMax) * (TREND_HEIGHT - TREND_PAD_TOP - TREND_PAD_BOTTOM);
    const svg = doc.createElementNS(SVG_NAMESPACE, "svg");
    svg.setAttribute("viewBox", `0 0 ${TREND_WIDTH} ${TREND_HEIGHT}`);
    svg.setAttribute("role", "img");
    const articlesTotal = series.reduce((sum, point) => sum + point.articles, 0);
    const draftsTotal = series.reduce((sum, point) => sum + point.drafts, 0);
    svg.setAttribute("aria-label", `近 ${range} 日每日写作动作趋势：文章 ${number.format(articlesTotal)} 次，草稿 ${number.format(draftsTotal)} 次`);
    for (let tick = 4; tick >= 1; tick--) {
      const line = doc.createElementNS(SVG_NAMESPACE, "line");
      const y1 = round(y((yMax / 4) * tick));
      line.setAttribute("x1", "0"); line.setAttribute("x2", String(TREND_WIDTH));
      line.setAttribute("y1", String(y1)); line.setAttribute("y2", String(y1));
      line.setAttribute("class", "activity-trend-grid");
      line.setAttribute("aria-hidden", "true");
      svg.append(line);
    }
    for (const [seriesClass, pick] of [
      ["activity-trend-articles", (point: { articles: number }) => point.articles],
      ["activity-trend-drafts", (point: { drafts: number }) => point.drafts],
    ] as const) {
      const path = doc.createElementNS(SVG_NAMESPACE, "path");
      path.setAttribute("d", smoothPath(series.map((point, index) => [x(index), y(pick(point))] as const)));
      path.setAttribute("class", seriesClass);
      path.setAttribute("fill", "none");
      path.setAttribute("aria-hidden", "true");
      svg.append(path);
    }
    container.append(svg);
    const labels = element("div", "activity-trend-labels");
    const step = Math.max(1, Math.ceil(series.length / 6));
    for (const [index, point] of series.entries()) {
      if (index % step) continue;
      const label = element("span", "");
      label.textContent = longDate(point.date);
      labels.append(label);
    }
    const lastPoint = series.at(-1);
    if (lastPoint && (series.length - 1) % step) {
      const label = element("span", "");
      label.textContent = longDate(lastPoint.date);
      labels.append(label);
    }
    container.append(labels);
  }

  function renderAll() {
    renderHeatmap();
    renderSummary();
    renderTrend();
  }

  function bindSwitch(selector: string, apply: (value: string) => void) {
    const group = $(selector);
    for (const button of group.querySelectorAll<HTMLButtonElement>("button")) {
      button.addEventListener("click", () => {
        if (button.getAttribute("aria-pressed") === "true") return;
        apply(button.dataset.activityMode ?? button.dataset.activityRange ?? "");
        for (const other of group.querySelectorAll<HTMLButtonElement>("button")) other.setAttribute("aria-pressed", String(other === button));
        renderAll();
      });
    }
  }

  async function refresh() {
    const current = ++revision;
    try {
      const result = await load();
      if (current !== revision) return;
      data = result;
      renderAll();
    } catch {
      // Keep showing the previous activity view; the dashboard reports load failures.
    }
  }

  bindSwitch("#dashboard-activity-modes", value => { if (value === "day" || value === "week" || value === "cumulative") mode = value; });
  bindSwitch("#dashboard-activity-ranges", value => { const parsed = Number(value); if (parsed === 7 || parsed === 30) range = parsed; });
  return { refresh };
}
