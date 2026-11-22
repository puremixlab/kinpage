import { expect, test } from "bun:test";
import { Window } from "happy-dom";
import { createWritingActivity } from "../src/writing-activity";
import type { DailyActivity } from "../src/mygo";

const tick = () => Bun.sleep(5);
const key = (offset: number) => {
  const today = new Date();
  const day = new Date(today.getFullYear(), today.getMonth(), today.getDate() - offset);
  return `${day.getFullYear()}-${String(day.getMonth() + 1).padStart(2, "0")}-${String(day.getDate()).padStart(2, "0")}`;
};
const activity: DailyActivity[] = [
  { date: key(1), articles: 1, drafts: 0 },
  { date: key(0), articles: 2, drafts: 4 },
];

async function fixture(load: () => Promise<DailyActivity[]> = async () => []) {
  const window = new Window();
  window.document.write(await Bun.file("index.html").text());
  const section = window.document.querySelector<HTMLElement>("#dashboard-activity")!;
  const view = createWritingActivity(section, load);
  return { view, section, window, $: <T extends Element = HTMLElement>(s: string) => section.querySelector<T>(s)! };
}

test("daily heatmap renders levels, tooltips, inline summary and trend series", async () => {
  const { view, section, $ } = await fixture(async () => activity);
  await view.refresh(); await tick();
  expect(section.querySelectorAll(".activity-grid > .activity-grid-column").length).toBe(53);
  expect(section.querySelectorAll(".activity-cell[data-level='4']").length).toBe(1);
  expect(section.querySelectorAll(".activity-cell[data-level='1']").length).toBe(1);
  const busiest = section.querySelector<HTMLElement>(".activity-cell[data-level='4']");
  expect(busiest!.title).toContain("6 次写作动作");
  expect($("#dashboard-activity-summary").textContent).toBe("近一年 7 次写作动作 · 连续 2 天");
  expect($("#dashboard-activity-trend-card").hidden).toBe(false);
  expect($("#dashboard-activity-range-heading").hidden).toBe(false);
  expect($("#dashboard-activity-trend svg path.activity-trend-articles")).not.toBeNull();
  expect($("#dashboard-activity-trend svg path.activity-trend-drafts")).not.toBeNull();
  expect($("#dashboard-activity-trend svg").getAttribute("aria-label")).toContain("文章 3 次");
  expect(section.querySelectorAll(".activity-trend-labels span").length).toBe(4);
});

test("activity modes switch between daily, weekly and cumulative grids", async () => {
  const { view, section, $ } = await fixture(async () => activity);
  await view.refresh(); await tick();
  $<HTMLButtonElement>("[data-activity-mode='week']").click();
  expect($("[data-activity-mode='week']").getAttribute("aria-pressed")).toBe("true");
  expect($("[data-activity-mode='day']").getAttribute("aria-pressed")).toBe("false");
  expect(section.querySelectorAll(".activity-grid-week > .activity-grid-column").length).toBe(53);
  expect(section.querySelectorAll(".activity-grid-week .activity-cell").length).toBe(53);
  const current = section.querySelector<HTMLElement>(".activity-grid-week .activity-cell[data-level='4']");
  expect(current!.title).toContain("当周 · 7 次写作动作");
  $<HTMLButtonElement>("[data-activity-mode='cumulative']").click();
  const cumulative = section.querySelector<HTMLElement>(".activity-grid-week .activity-cell[data-level='4']");
  expect(cumulative!.title).toContain("截至本周累计");
  expect($("#dashboard-activity-summary").textContent).toBe("全部时间共 7 次写作动作 · 覆盖 2 天");
  $<HTMLButtonElement>("[data-activity-mode='day']").click();
  expect(section.querySelectorAll(".activity-grid > .activity-grid-column").length).toBe(53);
});

test("trend range switch re-renders the daily series", async () => {
  const { view, section, $ } = await fixture(async () => activity);
  await view.refresh(); await tick();
  $<HTMLButtonElement>("[data-activity-range='30']").click();
  expect($("[data-activity-range='30']").getAttribute("aria-pressed")).toBe("true");
  expect($("[data-activity-range='7']").getAttribute("aria-pressed")).toBe("false");
  expect(section.querySelectorAll(".activity-trend-labels span").length).toBe(7);
  expect($("#dashboard-activity-trend svg").getAttribute("aria-label")).toContain("近 30 日");
});

test("empty library shows guidance and hides the trend chart", async () => {
  const { view, $ } = await fixture();
  await view.refresh(); await tick();
  expect($("#dashboard-activity-heatmap .dashboard-list-empty").textContent).toContain("还没有写作记录");
  expect($("#dashboard-activity-trend-card").hidden).toBe(true);
  expect($("#dashboard-activity-range-heading").hidden).toBe(true);
  expect($("#dashboard-activity-summary").hidden).toBe(true);
});

test("late activity responses cannot overwrite a newer view", async () => {
  let calls = 0;
  let resolveFirst!: (value: DailyActivity[]) => void;
  const { view, $ } = await fixture(() => {
    calls++;
    if (calls === 1) return new Promise<DailyActivity[]>(resolve => { resolveFirst = resolve; });
    return Promise.resolve(activity);
  });
  const pending = view.refresh(); await tick();
  await view.refresh(); await tick();
  resolveFirst([{ date: key(0), articles: 9, drafts: 9 }]); await tick();
  await pending;
  expect($("#dashboard-activity-summary").textContent).toBe("近一年 7 次写作动作 · 连续 2 天");
});

test("a failed refresh retains the last rendered activity", async () => {
  let fail = false;
  const { view, $ } = await fixture(async () => { if (fail) throw new Error("offline"); return activity; });
  await view.refresh(); await tick();
  fail = true; await view.refresh(); await tick();
  expect($("#dashboard-activity-summary").textContent).toBe("近一年 7 次写作动作 · 连续 2 天");
  expect($("#dashboard-activity-trend-card").hidden).toBe(false);
});
