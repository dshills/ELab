import {defineConfig,devices} from '@playwright/test';

export default defineConfig({
 testDir: './tests',
 fullyParallel: false,
 workers: 1,
 forbidOnly: !!process.env.CI,
 retries: 0,
 timeout: 60_000,
 expect: {timeout: 15_000},
 outputDir: '../../build/web-browser/results',
 reporter: [['list'],['html',{outputFolder:'../../build/web-browser/report',open:'never'}]],
 use: {
  baseURL: 'http://127.0.0.1:18788',
  viewport: {width:1440,height:1000},
  trace: 'retain-on-failure',
  screenshot: 'only-on-failure',
  video: 'off',
 },
 projects: [
  {name:'chromium',use:{...devices['Desktop Chrome']}},
  {name:'firefox',use:{...devices['Desktop Firefox']}},
  {name:'webkit',use:{...devices['Desktop Safari']}},
 ],
 webServer: {
  command: 'node host.mjs',
  url: 'http://127.0.0.1:18788/api/catalog',
  reuseExistingServer: false,
  timeout: 30_000,
  gracefulShutdown: {signal:'SIGTERM',timeout:40_000},
 },
});
