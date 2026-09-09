#!/usr/bin/env python3
"""
Test program for libzclone
"""

import os
import subprocess
import unittest
from zclone import *

class TestZclone(unittest.TestCase):
    """TestSuite for zclone python module"""
    shared_object = "libzclone.so"

    @classmethod
    def setUpClass(cls):
        super(TestZclone, cls).setUpClass()
        cls.shared_object = "./libzclone.so"
        Zclone.build(cls.shared_object)
        cls.zclone = Zclone(shared_object=cls.shared_object)

    @classmethod
    def tearDownClass(cls):
        cls.zclone.close()
        os.remove(cls.shared_object)
        super(TestZclone, cls).tearDownClass()

    def test_rpc(self):
        o = self.zclone.rpc("rc/noop", a=42, b="string", c=[1234])
        self.assertEqual(dict(a=42, b="string", c=[1234]), o)

    def test_rpc_error(self):
        try:
            o = self.zclone.rpc("rc/error", a=42, b="string", c=[1234])
        except ZcloneException as e:
            self.assertEqual(e.status, 500)
            self.assertTrue(e.output["error"].startswith("arbitrary error"))
        else:
            raise ValueError("Expecting exception")

if __name__ == '__main__':
    unittest.main()
