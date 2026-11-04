// These activities guide exploration; advancing a step is not a learning score.
export const activities={
 led:{example:'led',title:'Make a light change',steps:[
 ['Meet the circuit','This ready-made circuit has a supply, a resistor and an LED (a small light). Voltage, measured in volts (V), pushes charge around the circuit. Current, measured in amperes (A), is the flow of charge.'],
 ['Find the current','Select the LED on the drawing. Read its current in the inspector. A value in mA means milliamperes: 1 mA is one thousandth of an ampere. You do not need to calculate it.'],
 ['Change one thing','Select R1, the resistor. In Resistance, replace 1 kΩ with 2 kΩ, then choose Apply. Resistance limits current; kΩ means thousands of ohms. Leave the other values alone so the comparison is fair.'],
 ['Compare and keep your work','Select the LED again. Its current should be lower. Brightness is an illustration, not a prediction of a real LED rating. Choose Save project and give this circuit a name. You can find it again in Projects.']
 ]},
 rc:{example:'rc',title:'Watch a capacitor charge',steps:[
 ['Meet the circuit','R means resistor and C means capacitor. A capacitor stores electrical charge. This example lets you watch its voltage change over time in the Scope below the drawing.'],
 ['Watch the trace','Choose Restart simulation, then Resume if paused. The voltage rises smoothly rather than jumping immediately. Pause holds the displayed time; Resume continues it.'],
 ['Slow the change','Select C1. Change Capacitance from 1 µF to 2 µF and choose Apply. µF means microfarads, a unit of capacitance. Restart simulation to compare from the same starting point. The larger capacitor should take longer to charge.'],
 ['Keep an experiment','Choose Record, review its duration and choose Start recording in the dialog. When it completes, use the Scope playback controls to revisit captured samples. Save project to keep your named circuit; keep the recording if you want it included when reopening.']
 ]},
 bridge:{example:'discrete-hbridge',title:'See current reverse',steps:[
 ['Meet the circuit','An H-bridge uses four switches (here, MOSFET transistors) to reverse the voltage across a load. RL is the load in this example. You do not need to understand every transistor to explore the result.'],
 ['Watch the load','Select RL. Its voltage and current are signed: a minus sign means the direction is opposite to the measurement reference, not that the measurement is broken.'],
 ['Capture both directions','Choose Record. Set Duration to 40 ms and Sample interval to 1 ms, then record. This captures more than one direction interval without needing to click Step repeatedly.'],
 ['Compare the result','Use the completed recording and its Scope controls to inspect the load voltage on either side of 20 ms. It changes sign as the bridge reverses. Save the circuit and keep the recording to reopen this experiment later.']
 ]}
};
export function learningGuide(api){
 const panel=document.getElementById('learning-guide'),title=document.getElementById('learning-title'),step=document.getElementById('learning-step'),body=document.getElementById('learning-body');
 const previous=document.getElementById('learning-previous'),next=document.getElementById('learning-next');let lesson=null,index=0;
 function close(){lesson=null;panel.hidden=true;}
 function render(){title.textContent=lesson.title;step.textContent=`Step ${index+1} of ${lesson.steps.length}: ${lesson.steps[index][0]}`;body.textContent=lesson.steps[index][1];previous.disabled=index===0;next.textContent=index===lesson.steps.length-1?'Finish activity':'Next step';panel.hidden=false;}
 async function start(id){const chosen=activities[id];if(!chosen)return;document.getElementById('help-dialog').close();try{if(await api.load(chosen.example)===false)return;lesson=chosen;index=0;render();next.focus();}catch(e){api.toast(e.message,true);}}
 previous.onclick=()=>{if(lesson&&index>0){index--;render();}};
 next.onclick=()=>{if(!lesson)return;if(index===lesson.steps.length-1){close();document.getElementById('schematic').focus();}else{index++;render();}};
 document.getElementById('learning-close').onclick=()=>{close();document.getElementById('schematic').focus();};
 document.querySelectorAll('[data-lesson]').forEach(button=>button.onclick=()=>start(button.dataset.lesson));
 return {update(state){if(lesson&&state.example?.info?.id!==lesson.example)close();}};
}
