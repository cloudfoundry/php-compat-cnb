<?php
// src/Controller/LuckyController.php
namespace App\Controller;

use Symfony\Component\HttpFoundation\Response;

class LuckyController
{
    public function number()
    {
        $number = 42;

        return new Response(
            '<html><body>Lucky number: '.$number.'</body></html>'
        );
    }
}
