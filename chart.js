async function updateChart() {
    let symbol = document.getElementById('symbol').value;
    let tradeEnterDate = document.getElementById('tradeEnterDate').value;
    let buyPrice = document.getElementById('buyPrice').value;
    let exitPrice = document.getElementById('exitPrice').value;
    let tradeExitDate = document.getElementById('tradeExitDate').value;
    let exitPrice2 = document.getElementById('exitPrice2').value;
    let tradeExitDate2 = document.getElementById('tradeExitDate2').value;

    // Fetch data for daily chart
    let respDaily = await fetch('/data?symbol=' + symbol + '&tradeEnterDate=' + tradeEnterDate + '&tradeExitDate=' + tradeExitDate + '&buyPrice=' + buyPrice + '&exitPrice=' + exitPrice + '&tradeExitDate2=' + tradeExitDate2 + '&exitPrice2=' + exitPrice2 + '&showTrades=true');
    let dataDaily = await respDaily.json();
    renderChart('chartDaily', dataDaily);

    // Fetch data for weekly chart
    let respWeekly = await fetch('/data?symbol=' + symbol + '&tradeEnterDate=' + tradeEnterDate + '&tradeExitDate=' + tradeExitDate + '&buyPrice=' + buyPrice + '&exitPrice=' + exitPrice + '&tradeExitDate2=' + tradeExitDate2 + '&exitPrice2=' + exitPrice2 + '&timeFrame=weekly' + '&showTrades=true');
    let dataWeekly = await respWeekly.json();
    renderChart('chartWeekly', dataWeekly);

    // Fetch data for monthly chart
    let respMonthly = await fetch('/data?symbol=' + symbol + '&tradeEnterDate=' + tradeEnterDate + '&tradeExitDate=' + tradeExitDate + '&buyPrice=' + buyPrice + '&exitPrice=' + exitPrice + '&tradeExitDate2=' + tradeExitDate2 + '&exitPrice2=' + exitPrice2 + '&timeFrame=monthly' + '&showTrades=true');
    let dataMonthly = await respMonthly.json();
    renderChart('chartMonthly', dataMonthly);

    // Fetch data for SP500 chart
    let respSP500 = await fetch('/data?symbol=spy' + '&tradeEnterDate=' + tradeEnterDate + '&tradeExitDate=' + tradeExitDate + '&buyPrice=' + buyPrice + '&exitPrice=' + exitPrice + '&tradeExitDate2=' + tradeExitDate2 + '&exitPrice2=' + exitPrice2 + '&timeFrame=daily' + '&showTrades=true');
    let dataSP500 = await respSP500.json();
    renderChart('chartSP500', dataSP500);
}

function renderChart(elementId, data) {
    var chartContainer = document.querySelector("#" + elementId);
    chartContainer.innerHTML = '';

    var pointAnnotations = [];
    var xAxisAnnotations = [];



    // Check and process point annotations if they exist
    if (data.annotations.points && data.annotations.points.length > 0) {
        pointAnnotations = data.annotations.points.map(point => ({
            x: new Date(point.x).getTime(),
            y: point.y,
            marker: {
                size: point.marker.size,
                fillColor: point.marker.fillColor,
                strokeColor: point.marker.strokeColor,
                radius: 2
            },
            label: {
                borderColor: point.label.borderColor,
                text: point.label.text,
                textAnchor: 'middle',
                offsetX: 0,
                offsetY: -10,
                style: {
                    background: point.label.style.background,
                    color: point.label.style.color,
                    fontSize: '12px',
                    padding: {
                        left: 5,
                        right: 5,
                        top: 2,
                        bottom: 2
                    }
                }
            }
        }));
    }

    // Check and process x-axis annotations if they exist
    if (data.annotations.xaxis && data.annotations.xaxis.length > 0) {
        xAxisAnnotations = data.annotations.xaxis.map(range => ({
            x: new Date(range.x).getTime(),
            x2: new Date(range.x2).getTime(),
            fillColor: range.fillColor,
            label: {
                text: range.label.text
            }
        }));
    }

    var options = {
        chart: {
            type: 'candlestick',
            height: 400,
            toolbar: {
                autoSelected: 'pan',
                show: true
            },
            zoom: {
                enabled: true
            },
        },
        series: [{
            data: data.series[0].data
        }],
        title: {
            text: data.series[0].name,
            align: 'left'
        },
        xaxis: {
            type: 'datetime',
        },
        yaxis: {
            tooltip: {
                enabled: true
            }
        },
        annotations: {
            xaxis: xAxisAnnotations,
            points: pointAnnotations
        }
    };

    var chart = new ApexCharts(chartContainer, options);
    chart.render();
}

document.addEventListener('DOMContentLoaded', function () {
    document.getElementById('generate').onclick = updateChart;
});
