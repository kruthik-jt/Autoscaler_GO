<? phpini_set ( ' max_execution_time ', 600) ;
$x = 0.0001;
$arg = $_GET [ ' bytes '];
$bytes = ( int ) $arg ;
for ( $i = 0; $i <= $bytes ; $i ++) {
  $x += sqrt ( $x ) ;
}
echo $bytes ;
? >
