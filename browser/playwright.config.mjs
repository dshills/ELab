import {defineConfig,devices} from '@playwright/test';

// Split each engine's tests across disposable hosts to stay under the workspace cap.
const experimentTests=/[\\/](?:compare-change|schematic-replay|troubleshoot)\.spec\.mjs$/;
const largeCircuitTests=/[\\/](?:circuit-capacity|sheets)\.spec\.mjs$/;
const engines=[
 {name:'chromium',device:'Desktop Chrome',port:18788,largeCircuitPort:18791},
 {name:'firefox',device:'Desktop Firefox',port:18789,largeCircuitPort:18792},
 {name:'webkit',device:'Desktop Safari',port:18790,largeCircuitPort:18793},
];
const groups=engines.flatMap(({name,device,port,largeCircuitPort})=>[
 {name,device,port,testIgnore:[largeCircuitTests,experimentTests]},
 {name:`${name}-large-circuits`,device,port:largeCircuitPort,testMatch:[largeCircuitTests,experimentTests]},
]);

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
 projects: groups.map(({device,port,...project})=>({
  ...project,
  use: {...devices[device],baseURL:`http://127.0.0.1:${port}`},
 })),
 webServer: groups.map(({port})=>({
  command: 'node host.mjs '+port,
  url: `http://127.0.0.1:${port}/`,
  reuseExistingServer: false,
  timeout: 30_000,
  gracefulShutdown: {signal:'SIGTERM',timeout:40_000},
 })),
});
