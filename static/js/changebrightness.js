light = true
function changebrightness() {
    light = !light;
    if (light) {
        $("body")[0].className = "mdui-theme-primary-light-blue mdui-theme-accent-blue"
        $('.material-icons').first()[0].innerHTML = "brightness_2"
        return;
    }
    $("body")[0].className = "mdui-theme-primary-light-blue mdui-theme-accent-blue mdui-theme-layout-dark"
    $('.material-icons').first()[0].innerHTML = "wb_sunny"
}