"""Plot the exported measurements with ReportLab's vector line-chart renderer."""
import json
from pathlib import Path
from reportlab.graphics.shapes import Drawing, String, Rect, Group
from reportlab.graphics.charts.lineplots import LinePlot
from reportlab.graphics import renderSVG
from reportlab.lib.colors import HexColor

folder = Path(__file__).resolve().parent
rows = json.loads((folder / 'class-ab-response.json').read_text())['rows']
drawing = Drawing(960, 640)
drawing.add(Rect(0, 0, 960, 640, fillColor=HexColor('#ffffff'), strokeColor=None))
blue, amber = HexColor('#1573b8'), HexColor('#c56816')
ink, muted = HexColor('#172936'), HexColor('#506774')
drawing.add(String(60, 605, 'Class-AB amplifier: measured simulation', fontName='Helvetica-Bold', fontSize=19, fillColor=ink))
drawing.add(String(60, 581, '10 Hz input | ±5 V supplies | 100 Ω load | feedback gain 2', fontName='Helvetica', fontSize=12, fillColor=muted))

def panel(bottom, indices, factors, low, high, unit):
    plot = LinePlot()
    plot.x, plot.y, plot.width, plot.height = 85, bottom, 815, 185
    plot.data = [[(row['time_s'] * 1000, row['values'][index] * factor) for row in rows]
                 for index, factor in zip(indices, factors)]
    plot.joinedLines = 1
    plot.lines[0].strokeColor = blue
    plot.lines[1].strokeColor = amber
    for i in range(2):
        plot.lines[i].strokeWidth = 1.8
    plot.xValueAxis.valueMin, plot.xValueAxis.valueMax = 0, 200
    plot.xValueAxis.valueSteps = list(range(0, 201, 25))
    plot.xValueAxis.joinAxisMode = "bottom"
    plot.yValueAxis.valueMin, plot.yValueAxis.valueMax = low, high
    for axis in (plot.xValueAxis, plot.yValueAxis):
        axis.labels.fontName = 'Helvetica'
        axis.labels.fontSize = 10
        axis.labels.fillColor = ink
        axis.strokeColor = muted
        axis.visibleGrid = True
        axis.gridStrokeColor = HexColor('#dfe7eb')
        axis.gridStrokeWidth = .4
    drawing.add(plot)
    label = Group(String(0, 0, unit, textAnchor='middle', fontName='Helvetica', fontSize=11, fillColor=ink))
    label.transform = (0, 1, -1, 0, 25, bottom + 92)
    drawing.add(label)
    drawing.add(String(492, bottom - 32, 'Time (ms)', textAnchor='middle', fontName='Helvetica', fontSize=11, fillColor=ink))

panel(345, (0, 1), (1, 1), -1.1, 1.1, 'Voltage (V)')
drawing.add(String(100, 548, 'Input: 0.5 V peak', fontName='Helvetica', fontSize=11, fillColor=blue))
drawing.add(String(360, 548, 'Output: 0.99997 V peak', fontName='Helvetica', fontSize=11, fillColor=amber))
panel(80, (2, 3), (1000, -1000), 0, 10.5, 'Collector-current magnitude (mA)')
drawing.add(String(100, 284, 'Q1: sources the positive half-cycle', fontName='Helvetica', fontSize=11, fillColor=blue))
drawing.add(String(475, 284, '|Q2|: sinks the negative half-cycle', fontName='Helvetica', fontSize=11, fillColor=amber))
drawing.add(String(60, 17, 'Generic educational models. Bandwidth, slew rate, device ratings and heating are excluded.', fontName='Helvetica', fontSize=10, fillColor=muted))
renderSVG.drawToFile(drawing, str(folder / 'response.svg'))
