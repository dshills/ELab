import {defineConfig,devices} from '@playwright/test';

const ports={chromium:18788,firefox:18789,webkit:18790};
// Independent hosts prevent a combined run exhausting the production session cap.
const hosts=Object.entries(ports);

export default defineConfig({
 testDir: './tests',
 fullyParallel: false,
 workers: 1,
 forbidOnly: !!process.env.CI,
 retries: 0,
 timeout: 60_000,
 expect: {timeout: 15_000},
 outputDir: '../build/web-browser/results',
 reporter: [['list'],['html',{outputFolder:'../build/web-browser/report',open:'never'}]],
 use: {
  viewport: {width:1440,height:1000},
  trace: 'retain-on-failure',
  screenshot: 'only-on-failure',
  video: 'off',
 },
 projects: [
  {name:'chromium',use:{...devices['Desktop Chrome'],baseURL:`http://127.0.0.1:${ports.chromium}`}},
  {name:'firefox',use:{...devices['Desktop Firefox'],baseURL:`http://127.0.0.1:${ports.firefox}`}},
  {name:'webkit',use:{...devices['Desktop Safari'],baseURL:`http://127.0.0.1:${ports.webkit}`}},
 ],
 webServer: hosts.map(([,port])=>({
  command: 'node host.mjs '+port,
  url: `http://127.0.0.1:${port}/api/catalog`,
  reuseExistingServer: false,
  timeout: 30_000,
  gracefulShutdown: {signal:'SIGTERM',timeout:40_000},
 })),
});
