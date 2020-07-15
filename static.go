package struc2frm

const staticTplMainHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Department shuffler</title>
    <style>
        * {
            font-size: 14px;
            font-family: monospace;
        }
        div {
            padding: 4px;
            margin:  4px;
        }

        div.res {
            float:left;
            margin-right: 20px;
            border: 1px solid #aaaaaa;
            min-width: 250px;
        }
    </style>
</head>

<body>

        %v
        
        <br>
        <br>
        <!-- result bins -->
        %v


</body>
</html>`

const staticDefaultCSS  = `
div.struc2frm {
    padding: 4px;
    margin:  4px;
    border: 1px solid #aaa;
    border-radius: 6px;
}

div.struc2frm  h3 {
    padding: 4px;
    margin:  4px;
} 

div.struc2frm  input, 
div.struc2frm  textarea, 
div.struc2frm  select, 
div.struc2frm  button, 
div.struc2frm  label {
    padding: 4px;
    margin:  4px;
}
div.struc2frm  label {
    display: inline-block;
    vertical-align: middle;
    margin-top: 1px;
    text-align: right;
}
div.struc2frm  span.postlabel {
    display: inline-block;
    vertical-align: middle;
    font-size: 90%;
    position: relative;
    top: -3px;
    margin-left: 4px;
    max-width: 40px;
    line-height: 90%;
}

div.struc2frm  div.separator {
    height: 1px; 
    border-top: 1px solid #aaa; 

    padding: 0;
    margin: 0;
    margin-top: 4px;
    margin-bottom: 4px;
}

div.struc2frm  fieldset {
    border: 1px solid #aaa; 
    padding: 4px;
    margin:  14px 4px;
    border-radius: 8px;

}
div.struc2frm  legend {
    font-size: 90%;
    margin-left: 8px; 

    color: #444; 
    border: 1px solid #aaa; 
    padding: 0px 8px;
    border-radius: 5px;
}

div.struc2frm  button[type=submit],
div.struc2frm  input[type=submit]
{
    margin-left: 186px; /*some default*/
    width: 280px;
    height: 40px;
    padding: 4px 16px;
    margin-top: 12px;
    margin-bottom: 8px;
    border-radius: 6px;
}

/* Hiding spinners for integers/floats  */
/* stackoverflow.com/questions/3790935/ */
input[type="number"]::-webkit-outer-spin-button,
input[type="number"]::-webkit-inner-spin-button {
    -webkit-appearance: none;
    margin: 0;
}
input[type="number"] {
    -moz-appearance: textfield;
}

/* if s2f.Indent == 0   -   set values by CSS */
/* ========================================== */

/* Smartphones (portrait and landscape) */
@media screen and (max-width: 1023px){

    div.struc2frm  label {
        min-width: 90px;
    }
    div.struc2frm  h3 {
        margin-left: 106px;
    }
    div.struc2frm  button[type=submit],
    div.struc2frm  input[type=submit]
    {
        margin-left: 106px;
    }

}



/* Desktops and laptops */
@media screen and (min-width: 1024px){

    div.struc2frm  label {
        min-width: 120px;
    }
    div.struc2frm  h3 {
        margin-left: 136px;
    }
    div.struc2frm  button[type=submit],
    div.struc2frm  input[type=submit]
    {
        margin-left: 136px;
    }

}

/* Large screens */
@media screen and (min-width: 1824px){

    div.struc2frm  label {
        min-width: 150px;
    }
    div.struc2frm  h3 {
        margin-left: 166px;
    }
    div.struc2frm  button[type=submit],
    div.struc2frm  input[type=submit]
    {
        margin-left: 166px;
    }

}


/* change specific inputs */
div.struc2frm label[for="time"] {
    min-width: 20px;
}
div.struc2frm select[name="department"] {
    background-color: darkkhaki;
}

.error-block {
    color: var(--clr-err);
}
`
