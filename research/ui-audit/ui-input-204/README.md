# Packaged ELab 0.181.3 /204 input evidence

Unmodified CUA window captures from the exact packaged executable identified by
[the receipt](../../../docs/validation/ui-input-204/native-ui.json). Native process
identity was checked separately against the release path. No owner file was edited.

1. Scope fast drag changed A from 3.8900529865125244 s to 4.3823458574181124 s;
right-click and Control-click preserved it. Divider drag changed 0.65 to 0.47680414.
The right rail was not yet painted in this immediate capture; later captures show it.
2. Inspector thumb returned to the top after wheel scrolling to lower pinned controls.
3. Sweep drag moved point 2 (0.833333333 V) to point 6 (4.16666667 V). Secondary
clicks preserved point 6. The synthetic sweep was unkept, restoring the clean template.
4. Secondary clicks kept four components; primary click placed a fifth, with no
false stale-input warning. The placement was subsequently undone.
5. Immediate capture after changing inspector section; some content has not painted.
6. Settled capture of the same clean four-component LED template, with no additional
UI input between captures 5 and 6. Autosave was restored; existing recovery stayed
listed with its original 20-component count and timestamp. App left open here.

The [baseline and preview captions](../ui-input-203/README.md) retain the first-pass
failures and rejected /203 candidate. They do not qualify the final executable.
