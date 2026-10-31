from pathlib import Path
import unittest
from unittest import mock
import macos_deployment as m

BASE='Load command 0\n cmd LC_BUILD_VERSION\n platform 1\n minos 27.0\n sdk 27.0\n'

class Deployment(unittest.TestCase):
 def test_universal_floor_uses_highest_slice_without_lowering_anything(self):
  def tool(args,**kwargs):
   self.assertEqual(kwargs['timeout'],20)
   if args[0]=='lipo':return 'arm64 x86_64 arm64e.x1'
   return BASE if args[2]=='arm64' else BASE.replace('minos 27.0','minos 26.0')
  with mock.patch.object(m,'check_child_output',side_effect=tool):r=m.inspect(Path('fixture'))
  self.assertEqual(r['minimum'],'27.0');self.assertEqual(r['architectures']['x86_64']['minimum'],'26.0')
  m.verify({'LSMinimumSystemVersion':'27.0'},r,r)
  for bad in ('26.0','28.0',None):
   with self.assertRaisesRegex(ValueError,'differs'):m.verify({'LSMinimumSystemVersion':bad},r,r)
 def test_legacy_unrecorded_and_partial_metadata_are_distinct(self):
  m.verify({},None,None)
  with self.assertRaisesRegex(ValueError,'unbound'):m.verify({'LSMinimumSystemVersion':'27.0'},None,None)
  with self.assertRaisesRegex(ValueError,'differs'):m.verify({'LSMinimumSystemVersion':'27.0'},{},dict(minimum='27.0'))
 def test_malformed_or_conflicting_load_metadata_refused(self):
  for bad in ('',BASE+BASE,BASE.replace('platform 1','platform 2'),BASE.replace('minos 27.0','minos 28.0'),BASE.replace('minos 27.0','minos nope')):
   with self.subTest(bad=bad),self.assertRaises(ValueError):m.parse(bad)
 def test_missing_or_duplicate_architectures_refused(self):
  for value in ('','arm64 arm64','../../escape',' '.join(['x'+str(i) for i in range(9)])):
   with mock.patch.object(m,'check_child_output',return_value=value),self.assertRaises(ValueError):m.inspect(Path('fixture'))

if __name__=='__main__':unittest.main()
