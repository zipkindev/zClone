<?php
/*
PHP interface to libzclone.so, using FFI ( Foreign Function Interface )

Create an zclone object

$rc = new Zclone( __DIR__ . '/libzclone.so' );

Then call rpc calls on it

    $rc->rpc( "config/listremotes", "{}" );

When finished, close it

    $rc->close();
*/

class Zclone {

    protected $zclone;
    private $out;

    public function __construct( $libshared )
    {
        $this->zclone = \FFI::cdef("
        struct ZcloneRPCResult {
            char* Output;
            int	Status;
        };        
        extern void ZcloneInitialize();
        extern void ZcloneFinalize();
        extern struct ZcloneRPCResult ZcloneRPC(char* method, char* input);
        extern void ZcloneFreeString(char* str);
        ", $libshared);
        $this->zclone->ZcloneInitialize();
    }

    public function rpc( $method, $input ): array
    {
        $this->out = $this->zclone->ZcloneRPC( $method, $input );
        $response = [
            'output' => \FFI::string( $this->out->Output ),
            'status' => $this->out->Status
        ];
        $this->zclone->ZcloneFreeString( $this->out->Output );
        return $response;
    }

    public function close( ): void
    {
        $this->zclone->ZcloneFinalize();
    }
}
