import test from 'node:test';
import assert from 'node:assert/strict';
import {schemaFields,readSchemaForm,resolveSchema,labelFor} from './static/forms.mjs';
const form=values=>({elements:{namedItem:name=>name in values?{value:String(values[name]),checked:values[name]===true}:null},querySelectorAll:()=>[]});
test('schema controls preserve native references, nullables and engineering quantities',()=>{
 const registry={'urn:model':{type:'object',properties:{value:{anyOf:[{type:'number'},{type:'string'}]},enabled:{type:'boolean'},version:{type:'string',const:'v1'}},required:['value']}};
 const schema={anyOf:[{$ref:'urn:model'},{type:'null'}]};
 // Reference in a nullable branch resolves before rendering its fields.
 const resolved=resolveSchema(schema,registry);
 assert.equal(resolved.type,'object');
 assert.deepEqual(readSchemaForm(form({value:'4.7 uF',enabled:true,version:'v1'}),schema,registry),{value:'4.7 uF',enabled:true,version:'v1'});
 const html=schemaFields(schema,{value:'4.7 uF',enabled:true},registry);
 assert.ok(!schemaFields({type:'object',properties:{unset:{type:'string'}}}).includes('[object Object]'));
 assert.match(html,/name="version" value="v1"/);assert.doesNotMatch(html,/select/);
});
test('nested native models return typed finite numbers and omit blank optional values',()=>{
 const schema={type:'object',properties:{parameters:{type:'object',properties:{ambient_c:{type:'number'},label:{type:'string'}},required:['ambient_c']}},required:['parameters']};
 assert.deepEqual(readSchemaForm(form({'parameters.ambient_c':40,'parameters.label':''}),schema),{parameters:{ambient_c:40}});
 assert.throws(()=>readSchemaForm(form({'parameters.ambient_c':'Infinity'}),schema),/finite/);
 assert.throws(()=>readSchemaForm(form({'parameters.ambient_c':''}),schema),/required/);
});
test('saved model text is escaped and SI labels remain readable',()=>{
 const html=schemaFields({type:'object',properties:{name:{type:'string'}}},{name:'"><img src=x onerror=alert(1)>'});
 assert.ok(!html.includes('<img'));assert.match(html,/&quot;&gt;&lt;img/);
 assert.equal(labelFor('thermal_resistance_k_per_w'),'Thermal resistance (K/W)');
 assert.equal(labelFor('ambient_c'),'Ambient temperature (°C)');
});
