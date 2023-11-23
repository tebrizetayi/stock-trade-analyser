function getQueryStringParam(param) {
    const urlParams = new URLSearchParams(window.location.search);
    return urlParams.get(param);
}


async function updateChart() {
    let queryParams = [
        'symbol', 'tradeEnterDate', 'buyPrice', 'exitPrice',
        'tradeExitDate', 'exitPrice2', 'tradeExitDate2'
    ].reduce((acc, param) => {
        let value = getQueryStringParam(param);
        if (value) acc.push(param + '=' + encodeURIComponent(value));
        return acc;
    }, []).join('&');

    // Append common parameters
    let commonParams = queryParams + '&showTrades=true';

    // Fetch and render charts with proper error handling
    try {
        let dataDaily = await fetchAndParseData('/data?' + commonParams + '&timeFrame=daily');
        renderChart('chartDaily', dataDaily);

        let dataWeekly = await fetchAndParseData('/data?' + commonParams + '&timeFrame=weekly');
        renderChart('chartWeekly', dataWeekly);

        let dataMonthly = await fetchAndParseData('/data?' + commonParams + '&timeFrame=monthly');
        renderChart('chartMonthly', dataMonthly);

        let dataSP500 = await fetchAndParseData('/data?symbol=spy&' + commonParams);
        renderChart('chartSP500', dataSP500);
    } catch (error) {
        console.error('Error fetching chart data:', error);
    }
}

async function fetchAndParseData(url) {
    let response = await fetch(url);
    if (!response.ok) throw new Error('Network response was not ok');
    return response.json();
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
    updateChart();
});