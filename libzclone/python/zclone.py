"""
Python interface to libzclone.so using ctypes

Create an zclone object

    zclone = Zclone(shared_object="/path/to/libzclone.so")

Then call rpc calls on it

    zclone.rpc("rc/noop", a=42, b="string", c=[1234])

When finished, close it

    zclone.close()
"""

__all__ = ('Zclone', 'ZcloneException')

import os
import json
import subprocess
from ctypes import *

class ZcloneRPCString(c_char_p):
    """
    This is a raw string from the C API

    With a plain c_char_p type, ctypes will replace it with a
    regular Python string object that cannot be used with
    ZcloneFreeString. Subclassing prevents it, while the string
    can still be retrieved from attribute value.
    """
    pass

class ZcloneRPCResult(Structure):
    """
    This is returned from the C API when calling ZcloneRPC
    """
    _fields_ = [("Output", ZcloneRPCString),
                ("Status", c_int)]

class ZcloneException(Exception):
    """
    Exception raised from zclone

    This will have the attributes:

    output - a dictionary from the call
    status - a status number
    """
    def __init__(self, output, status):
        self.output = output
        self.status = status
        message = self.output.get('error', 'Unknown zclone error')
        super().__init__(message)

class Zclone():
    """
    Interface to Zclone via libzclone.so

    Initialise with shared_object as the file path of libzclone.so
    """
    def __init__(self, shared_object=f"./libzclone{'.dll' if os.name == 'nt' else '.so'}"):
        self.zclone = CDLL(shared_object)
        self.zclone.ZcloneRPC.restype = ZcloneRPCResult
        self.zclone.ZcloneRPC.argtypes = (c_char_p, c_char_p)
        self.zclone.ZcloneFreeString.restype = None
        self.zclone.ZcloneFreeString.argtypes = (c_char_p,)
        self.zclone.ZcloneInitialize.restype = None
        self.zclone.ZcloneInitialize.argtypes = ()
        self.zclone.ZcloneFinalize.restype = None
        self.zclone.ZcloneFinalize.argtypes = ()
        self.zclone.ZcloneInitialize()
    def rpc(self, method, **kwargs):
        """
        Call an zclone RC API call with the kwargs given.

        The result will be a dictionary.

        If an exception is raised from zclone it will of type
        ZcloneException.
        """
        method = method.encode("utf-8")
        parameters = json.dumps(kwargs).encode("utf-8")
        resp = self.zclone.ZcloneRPC(method, parameters)
        output = json.loads(resp.Output.value.decode("utf-8"))
        self.zclone.ZcloneFreeString(resp.Output)
        status = resp.Status
        if status != 200:
            raise ZcloneException(output, status)
        return output
    def close(self):
        """
        Call to finish with the zclone connection
        """
        self.zclone.ZcloneFinalize()
        self.zclone = None
    @classmethod
    def build(cls, shared_object):
        """
        Builds zclone to shared_object if it doesn't already exist

        Requires go to be installed
        """
        if os.path.exists(shared_object):
            return
        print("Building "+shared_object)
        subprocess.check_call(["go", "build", "--buildmode=c-shared", "-o", shared_object, "zclone/libzclone"])
